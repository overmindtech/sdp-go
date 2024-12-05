package llm

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestNewAnthropicProvider(t *testing.T) {
	t.Parallel()

	key, ok := os.LookupEnv("ANTHROPIC_API_KEY")
	if !ok {
		t.Skip("ANTHROPIC_API_KEY not found")
	}

	provider := NewAnthropicProvider(key, anthropic.ModelClaude3_5SonnetLatest)
	ctx := context.Background()

	conversation, err := provider.NewConversation(ctx, "", []ToolImplementation{
		&secretTool,
		&weatherTool,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("without using the tools", func(t *testing.T) {
		response, err := conversation.SendMessage(ctx, "Reply with the word 'banana', nothing else")
		if err != nil {
			t.Fatal(err)
		}

		if response != "banana" {
			t.Errorf("expected the response 'banana', got '%v'", response)
		}
	})

	t.Run("with using the tools", func(t *testing.T) {
		response, err := conversation.SendMessage(ctx, "Call the test-tool with the secret 'banana' and find the weather for London. Return me the results from both")
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(response, "pie") {
			t.Errorf("expected the response to contain 'pie', but it didn't. Response: %v", response)
		}

		if !strings.Contains(response, "meatballs") {
			t.Errorf("expected the response to contain 'meatballs', but it didn't. Response: %v", response)
		}
	})
}
