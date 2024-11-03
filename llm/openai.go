package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/overmindtech/sdp-go/tracing"
	"github.com/sashabaranov/go-openai"
	"github.com/sourcegraph/conc/iter"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Creates a new OpenAI Provider. The user must pass in the API key, the model
// to use, and the name. The name is used in the subsequent names of all the
// assistants that are created
func NewOpenAIProvider(apiKey string, model string, name string, jsonMode bool) *openAIProvider {
	cfg := openai.DefaultConfig(apiKey)
	cfg.HTTPClient = otelhttp.DefaultClient

	return &openAIProvider{
		client:   openai.NewClientWithConfig(cfg),
		model:    model,
		name:     name,
		jsonMode: jsonMode,
	}
}

type openAIProvider struct {
	client   *openai.Client
	model    string
	name     string
	jsonMode bool
}

func (p *openAIProvider) NewConversation(ctx context.Context, systemPrompt string, tools []ToolImplementation) (Conversation, error) {
	cleanup := make(cleanupTasks, 0)

	// Format the tools in the required format
	assistantTools := []openai.AssistantTool{}
	for _, tool := range tools {
		assistantTools = append(assistantTools, openai.AssistantTool{
			Type: openai.AssistantToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.ToolName(),
				Description: tool.ToolDescription(),
				Parameters:  tool.InputSchema(),
				// Not sure if we need strict or not?
				// Strict: true,
			},
		})
	}

	var responseFormat any

	if p.jsonMode {
		responseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}

	assistant, err := p.client.CreateAssistant(ctx, openai.AssistantRequest{
		Model:          p.model,
		Name:           &p.name,
		Instructions:   &systemPrompt,
		Tools:          assistantTools,
		ResponseFormat: responseFormat,
	})
	if err != nil {
		return nil, err
	}

	cleanup = append(cleanup, func() error {
		// Delete the assistant if something goes wrong
		_, err := p.client.DeleteAssistant(ctx, assistant.ID)
		return err
	})

	thread, err := p.client.CreateThread(ctx, openai.ThreadRequest{})
	if err != nil {
		err = cleanup.Run(err)

		return nil, err
	}

	return &openAIConversation{
		client:    p.client,
		assistant: assistant,
		thread:    thread,
		tools:     tools,
	}, nil
}

type openAIConversation struct {
	client    *openai.Client
	assistant openai.Assistant
	thread    openai.Thread
	tools     []ToolImplementation
}

// Cleans up the assistant and thread from a conversation
func (c *openAIConversation) End(ctx context.Context) error {
	_, threadErr := c.client.DeleteThread(ctx, c.thread.ID)
	_, assistantErr := c.client.DeleteAssistant(ctx, c.assistant.ID)

	return errors.Join(threadErr, assistantErr)
}

func (c *openAIConversation) SendMessage(ctx context.Context, userMessage string) (string, error) {
	ctx, span := tracing.Tracer().Start(ctx, "SendMessage")
	defer span.End()

	span.SetAttributes(attribute.String("ovm.assistant.userMessage", userMessage))
	// These tasks will be run if we hit an error at any point. If the method is
	// successful these will be ignored
	cleanup := make(cleanupTasks, 0)

	// Add a message to the existing thread
	message, err := c.client.CreateMessage(ctx, c.thread.ID, openai.MessageRequest{
		Role:    openai.ChatMessageRoleUser,
		Content: userMessage,
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	// Add a cleanup task to delete the message so that this can be re-run
	cleanup = append(cleanup, func() error {
		// Delete the last message so we don't end up with duplicates
		_, err := c.client.DeleteMessage(ctx, c.thread.ID, message.ID)
		return err
	})

	// Run that thread
	run, err := c.client.CreateRun(ctx, c.thread.ID, openai.RunRequest{
		AssistantID: c.assistant.ID,
	})
	if err != nil {
		err = cleanup.Run(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	// Wait for the thread to be complete
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			// Cancel the run. Use a new context to do this since we know that
			// the existing one has run out
			cancelRunCtx, cancelRunCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelRunCancel()
			_, cancelErr := c.client.CancelRun(cancelRunCtx, c.thread.ID, run.ID)
			err = errors.Join(err, cancelErr)

			// Run the other cleanup tasks
			err = cleanup.Run(err)

			span.SetStatus(codes.Error, err.Error())
			return "", err
		case <-ticker.C:
			// Check to see if the run is done
			run, err = c.client.RetrieveRun(ctx, c.thread.ID, run.ID)
			if err != nil {
				err = cleanup.Run(err)
				span.SetStatus(codes.Error, err.Error())
				return "", err
			}

			// Capture data from the LLM
			rates := run.GetRateLimitHeaders()
			span.SetAttributes(
				attribute.String("ovm.openai.model", run.Model),
				attribute.String("ovm.openai.assistantID", run.AssistantID),
				attribute.String("ovm.openai.threadID", run.ThreadID),

				// usage Represents the total token usage per request to OpenAI.
				attribute.Int("ovm.openai.completionTokens", run.Usage.CompletionTokens),
				attribute.Int("ovm.openai.promptTokens", run.Usage.PromptTokens),
				attribute.Int("ovm.openai.totalTokens", run.Usage.TotalTokens), // totalTokens is the sum of completionTokens and promptTokens(inputTokens).

				attribute.Int("ovm.openai.reqs.limit", rates.LimitRequests),
				attribute.Int("ovm.openai.reqs.remaining", rates.RemainingRequests),
				attribute.String("ovm.openai.reqs.resetRelative", rates.ResetRequests.String()),
				attribute.String("ovm.openai.reqs.resetAbsolute", rates.ResetRequests.Time().String()),

				attribute.Int("ovm.openai.tokens.limit", rates.LimitTokens),
				attribute.Int("ovm.openai.tokens.remaining", rates.RemainingTokens),
				attribute.String("ovm.openai.tokens.resetRelative", rates.ResetTokens.String()),
				attribute.String("ovm.openai.tokens.resetAbsolute", rates.ResetTokens.Time().String()),
			)

			switch run.Status {
			case openai.RunStatusQueued, openai.RunStatusInProgress, openai.RunStatusCancelling:
				// Do nothing
			case openai.RunStatusRequiresAction:
				requiredAction := run.RequiredAction
				if requiredAction == nil {
					err = errors.New("tool call returned with no required actions")
					err = cleanup.Run(err)
					span.SetStatus(codes.Error, err.Error())
					return "", err
				}

				switch requiredAction.Type {
				case openai.RequiredActionTypeSubmitToolOutputs:
					// get the tool inputs from the openai run
					submitToolOutputs := requiredAction.SubmitToolOutputs
					if submitToolOutputs == nil {
						err = errors.New("tools were requested but SubmitToolOutputs was nil")
						err = cleanup.Run(err)
						span.SetStatus(codes.Error, err.Error())
						return "", err
					}

					// run the tools, with a cancel context and a 2min timeout
					toolsCtx, toolsCancel := context.WithTimeout(ctx, 2*time.Minute)
					defer toolsCancel()

					outputs := callToolsOpenAI(toolsCtx, c.tools, submitToolOutputs.ToolCalls)

					// submit the tool outputs to the openai run
					_, err := c.client.SubmitToolOutputs(ctx, run.ThreadID, run.ID, openai.SubmitToolOutputsRequest{
						ToolOutputs: outputs,
					})
					if err != nil {
						err = cleanup.Run(err)
						span.SetStatus(codes.Error, err.Error())
						return "", err
					}
				}
			case openai.RunStatusCompleted:
				// Get the last message
				limit := 1
				order := "desc"
				messages, err := c.client.ListMessage(ctx, run.ThreadID, &limit, &order, nil, nil, &run.ID)
				if err != nil {
					// TODO: We could probably have a better retry here then to
					// just give up

					// Clean up everything
					err = cleanup.Run(err)
					span.SetStatus(codes.Error, err.Error())
					return "", err
				}

				// Extract and return this message
				if len(messages.Messages) == 0 {
					err = errors.New("empty messages returned from API")
					span.SetStatus(codes.Error, err.Error())
					return "", err
				}
				if len(messages.Messages[0].Content) == 0 {
					err = errors.New("empty content returned from API")
					span.SetStatus(codes.Error, err.Error())
					return "", err
				}

				return messages.Messages[0].Content[0].Text.Value, nil
			case openai.RunStatusIncomplete:
				err = errors.New("run was incomplete due to a token limit")
				err = cleanup.Run(err)
				span.SetStatus(codes.Error, err.Error())
				return "", err
			case openai.RunStatusFailed, openai.RunStatusExpired, openai.RunStatusCancelled:
				err = fmt.Errorf("unexpected run status: %v", run.Status)
				err = cleanup.Run(err)
				span.SetStatus(codes.Error, err.Error())
				return "", err
			}
		}
	}
}

// Calls all required tools as part of the `submit_tool_outputs` phase of an
// OpenAI interaction in parallel. The results will be fed back to the OpenAI
// API via the supplied client. Returns a slice of tool outputs.
func callToolsOpenAI(ctx context.Context, tools []ToolImplementation, toolCalls []openai.ToolCall) []openai.ToolOutput {
	ctx, span := tracing.Tracer().Start(ctx, "CallTools")
	defer span.End()
	responses := make([]openai.ToolOutput, len(toolCalls))

	// Call the tools in parallel
	iter.ForEachIdx(toolCalls, func(i int, tc *openai.ToolCall) {
		// Find the requested tool form the set
		var relevantTool ToolImplementation
		for _, tool := range tools {
			if tc.Function.Name == tool.ToolName() {
				relevantTool = tool
				break
			}
		}

		// If the tool was not found, return an error
		if relevantTool == nil {
			responses[i] = openai.ToolOutput{
				ToolCallID: tc.ID,
				Output:     fmt.Sprintf("Error: tool %s not found", tc.Function.Name),
			}

			return
		}
		// Call the tool
		toolCtx, span := tracing.Tracer().Start(ctx, tc.Function.Name, trace.WithAttributes(
			attribute.String("ovm.assistant.toolParameters", tc.Function.Arguments),
		))
		output, err := relevantTool.Call(toolCtx, []byte(tc.Function.Arguments))
		// send outputs from the tool to tracing
		span.SetAttributes(
			attribute.String("ovm.assistant.toolOutput", output),
			attribute.String("ovm.assistant.toolError", fmt.Sprintf("%v", err)),
		)
		defer span.End()
		// If there was an error, return that
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			responses[i] = openai.ToolOutput{
				ToolCallID: tc.ID,
				Output:     fmt.Sprintf("Error calling %s: %s", tc.Function.Name, err),
			}

			return
		}
		// Return the output
		responses[i] = openai.ToolOutput{
			ToolCallID: tc.ID,
			Output:     output,
		}
	})

	return responses
}

type cleanupTasks []func() error

// Runs all cleanup tasks and appends all errors to the error that is passed in
func (c cleanupTasks) Run(err error) error {
	errs := []error{err}
	for _, task := range c {
		errs = append(errs, task())
	}

	return errors.Join(errs...)
}
