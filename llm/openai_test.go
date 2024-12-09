package llm

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
)

func TestNewOpenAIProvider(t *testing.T) {
	t.Parallel()
	key, exists := os.LookupEnv("OPENAI_API_KEY")
	if !exists {
		t.Skip("OPENAI_API_KEY not set")
	}

	openaiProvider := NewOpenAIProvider(key, "", openai.GPT4oMini, "", t.Name(), false)

	// Assert that the result matches the provider interface
	var _ Provider = openaiProvider

	ctx := context.Background()
	conversation, err := openaiProvider.NewConversation(ctx, "", []ToolImplementation{
		&secretTool,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := conversation.End(context.Background())
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("without calling any tools", func(t *testing.T) {
		response, err := conversation.SendMessage(ctx, "Respond with the word 'banana'")
		if err != nil {
			t.Error(err)
		}

		if response != "banana" {
			t.Errorf("expected 'banana' as a response, got '%v'", response)
		}
	})

	t.Run("calling the test tool", func(t *testing.T) {
		response, err := conversation.SendMessage(ctx, "Call the secret-tool with the secret 'banana'")
		if err != nil {
			t.Error(err)
		}

		if !strings.Contains(response, "pie") {
			t.Errorf("expected 'pie' as a response, got '%v'", response)
		}
	})

	t.Run("cancelling a run", func(t *testing.T) {
		errChan := make(chan error)
		ctx, cancel := context.WithCancel(ctx)

		go func() {
			_, err := conversation.SendMessage(ctx, "Respond with the work 'banana' 50 times")
			errChan <- err
		}()

		time.Sleep(1 * time.Second)
		cancel()

		err := <-errChan

		// Make sure the error is a context cancelled error
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Unexpected error %T %v", err, err.Error())
		}
	})
}

func TestCleanupTasks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cleanup := cleanupTasks{
		func(ctx context.Context) error {
			return ctx.Err()
		},
	}

	// I want to cancel the context *then* run the cleanup tasks, this *should*
	// return nothing since the context that the cleanup task runs in *hasn't*
	// been cancelled yet as it gets another 10 seconds to do its work
	cancel()
	err := cleanup.Run(ctx, nil)

	if err != nil {
		t.Error(err)
	}
}
