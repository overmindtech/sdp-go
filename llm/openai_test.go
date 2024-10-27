package llm

import (
	"context"
	"os"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestNewOpenAIProvider(t *testing.T) {
	key, exists := os.LookupEnv("OPENAI_API_KEY")
	if !exists {
		t.Fatal("OPENAI_API_KEY not set")
	}

	openaiProvider := NewOpenAIProvider(key, openai.GPT4oMini, t.Name(), false)

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

		if response != "pie" {
			t.Errorf("expected 'pie' as a response, got '%v'", response)
		}
	})
}
