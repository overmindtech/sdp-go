package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/overmindtech/sdp-go/tracing"
	"github.com/sourcegraph/conc/iter"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func NewAnthropicProvider(apiKey string, model anthropic.Model) *anthropicProvider {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(otelhttp.DefaultClient),
	)

	return &anthropicProvider{
		client: client,
		model:  model,
	}
}

type anthropicProvider struct {
	client *anthropic.Client
	model  anthropic.Model
}

func (p *anthropicProvider) NewConversation(ctx context.Context, systemPrompt string, tools []ToolImplementation) (Conversation, error) {
	var system []anthropic.TextBlockParam

	if systemPrompt != "" {
		system = []anthropic.TextBlockParam{
			anthropic.NewTextBlock(systemPrompt),
		}
	}

	return &anthropicConversation{
		client:   p.client,
		messages: make([]anthropic.MessageParam, 0),
		model:    p.model,
		tools:    tools,
		system:   system,
	}, nil
}

type anthropicConversation struct {
	client   *anthropic.Client
	messages []anthropic.MessageParam
	model    anthropic.Model
	tools    []ToolImplementation
	system   []anthropic.TextBlockParam
}

func (c *anthropicConversation) SendMessage(ctx context.Context, userMessage string) (string, error) {
	ctx, span := tracing.Tracer().Start(ctx, "SendMessage")
	defer span.End()

	span.SetAttributes(
		attribute.String("ovm.llm.userMessage", userMessage),
		attribute.String("ovm.llm.provider", "anthropic"),
	)

	// Construct the list of tools
	tools := make([]anthropic.ToolParam, len(c.tools))
	for i, tool := range c.tools {
		schema := tool.InputSchema()
		tools[i] = anthropic.ToolParam{
			Name:        anthropic.F(tool.ToolName()),
			Description: anthropic.F(tool.ToolDescription()),
			InputSchema: anthropic.F(interface{}(schema)),
		}
	}

	updatedMessages := append(c.messages, anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)))

	// Since Anthropic does a bit of "thinking" it behaves a bit differently to
	// OpenAI. OpenAI seems to do its thinking "in it's head" and you just get
	// the results, whereas with Anthropic the LLM "thinks" out loud and tells
	// you what it's doing. I like this, but it makes it unclear how to handle
	// the responses in a consistent way. For the moment I'm adding all of the
	// text together and returning that, but we might want to change that
	var assistantResponse string

	for {
		// Send the message to the LLM
		response, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.F(c.model),
			MaxTokens: anthropic.Int(8192),
			Messages:  anthropic.F(updatedMessages),
			Tools:     anthropic.F(tools),
			System:    anthropic.F(c.system),
		})
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return "", err
		}

		// Extract tracing information from the response
		span.SetAttributes(
			attribute.Int64("ovm.anthropic.usage.inputTokens", response.Usage.InputTokens),
			attribute.Int64("ovm.anthropic.usage.outputTokens", response.Usage.OutputTokens),
			attribute.String("ovm.anthropic.model", response.Model),
		)

		// Save the response
		updatedMessages = append(updatedMessages, response.ToParam())

		// Pull out the tool calls, then run them in parallel. If there aren't
		// any tool calls then we can assume that the assistant is finished
		toolCalls := make([]anthropic.ContentBlock, 0)
		for _, contentBlock := range response.Content {
			switch contentBlock.Type {
			case anthropic.ContentBlockTypeToolUse:
				toolCalls = append(toolCalls, contentBlock)
			case anthropic.ContentBlockTypeText:
				assistantResponse = strings.Join([]string{assistantResponse, contentBlock.Text}, "\n")
			}
		}

		if len(toolCalls) == 0 {
			// If there aren't any tools being called, just return the text form
			// the LLM, and save the messages that were generated as this was
			// successful
			c.messages = updatedMessages
			return strings.TrimPrefix(assistantResponse, "\n"), nil
		}

		toolResults := callToolsAnthropic(ctx, c.tools, toolCalls)

		// Add the responses to the messages
		updatedMessages = append(updatedMessages, anthropic.NewUserMessage(toolResults...))
	}
}

func callToolsAnthropic(ctx context.Context, tools []ToolImplementation, toolCalls []anthropic.ContentBlock) []anthropic.MessageParamContentUnion {
	ctx, span := tracing.Tracer().Start(ctx, "CallTools")
	defer span.End()
	responses := make([]anthropic.MessageParamContentUnion, len(toolCalls))

	iter.ForEachIdx(toolCalls, func(i int, block *anthropic.ContentBlock) {
		// Find the requested tool form the set
		var relevantTool ToolImplementation
		for _, tool := range tools {
			if block.Name == tool.ToolName() {
				relevantTool = tool
				break
			}
		}

		if relevantTool == nil {
			responses[i] = anthropic.NewToolResultBlock(
				block.ID,
				fmt.Sprintf("Error: tool %s not found", block.Name),
				true,
			)
		}

		// Call the tool
		toolCtx, span := tracing.Tracer().Start(ctx, block.Name, trace.WithAttributes(
			attribute.String("ovm.assistant.toolParameters", string(block.Input)),
		))
		output, err := relevantTool.Call(toolCtx, []byte(block.Input))
		// send outputs from the tool to tracing
		span.SetAttributes(
			attribute.String("ovm.assistant.toolOutput", output),
			attribute.String("ovm.assistant.toolError", fmt.Sprintf("%v", err)),
		)
		defer span.End()
		// If there was an error, return that
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true))
			responses[i] = anthropic.NewToolResultBlock(
				block.ID,
				fmt.Sprintf("Error calling %s: %s", block.Name, err),
				true,
			)

			return
		}

		// Return the output
		responses[i] = anthropic.NewToolResultBlock(
			block.ID,
			output,
			false,
		)
	})

	return responses
}

func (c *anthropicConversation) End(ctx context.Context) error {
	// There isn't any cleanup that we need to do with the Anthropic API since
	// the state is all stored locally rather than on their side
	return nil
}
