package llm

import (
	"context"
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// For the LLM interface, we will need to have a method or something that we can
// call once the process is over since there is cleanup to do. There will also
// be setup, though this potentially doesn't need to be a method. The
// implementation could just check to see if it's the first call and set things
// up if it is.
type Provider interface {
	// Creates a new conversation with the LLM. This will return a conversation
	// that automatically tracks messages back and forth. The user is
	// responsible for ending the conversation with `End()` to ensure that
	// resources are released
	NewConversation(ctx context.Context, systemPrompt string, tools []ToolImplementation) (Conversation, error)
}

// A conversation with the LLM. This is used to send messages to the LLM and
// receive responses. The conversation will automatically track the messages
// back and forth, so the user can call the `SendMessage()` method multiple
// times and the conversation will keep track of the state.
//
// The user is responsible for calling `End()` to ensure that resources are
// released
type Conversation interface {
	// Sends a new message to this conversation. This will wait for the LLM to
	// response and return the response as a string
	SendMessage(ctx context.Context, userMessage string) (string, error)

	// Ends the conversation. This does any required cleanup and should always
	// be deferred after calling `NewConversation()`
	End(ctx context.Context) error
}

// A tool that can be run as part of an LLM operation
type ToolImplementation interface {
	// The name of the tool
	ToolName() string

	// A textual description of what the tool does, and how and when to use it
	ToolDescription() string

	// The JSON schema for the inputs for this function. An example schema could
	// be:
	//
	// ```json
	// {
	//   "type": "object",
	//   "properties": {
	//     "location": {
	//       "type": "string",
	//       "description": "The city and country, eg. San Francisco, USA"
	//     },
	//     "format": { "type": "string", "enum": ["celsius", "fahrenheit"] }
	//   },
	//   "required": ["location", "format"]
	// }
	// ```
	InputSchema() *jsonschema.Schema

	// Call the tool with the given context and parameters. The parameters will
	// be provided as a JSON encoded string.
	Call(ctx context.Context, jsonInput []byte) (string, error)
}

// This type is a helper that implements the `ToolImplementation` interface and
// uses reflection to automatically determine the JSON schema, and to
// deserialise the inputs to the tool. It is recommended that you use this when
// implementing tools
//
// The InputType should be a struct that has been annotated with tags from the
// [jsonschema package](https://pkg.go.dev/github.com/invopop/jsonschema) for
// example:
//
// ```go
//
//	type WeatherToolInput struct {
//		Location string `json:"location" jsonschema_description:"The location that we should get the weather for"`
//		Units    string `json:"units,omitempty" jsonschema:"enum=fahrenheit,enum=celsius"`
//	}
//
// ```
type Tool[InputType any] struct {
	// The name of the tool. This must contains only A-z, 0-9, dashes and
	// underscores
	Name string
	// A description of the tool. This is what is passed to the LLM and should
	// give a clear a detailed description of what the tool does and how/when it
	// should be used. You do not need to describe the inputs here as the input
	// schema is determined automatically from the `InputType`
	Description string
	Func        func(ctx context.Context, inputData InputType) (string, error)
}

// Returns the name of the tool
func (t *Tool[InputData]) ToolName() string {
	return t.Name
}

// Returns the tool's description
func (t *Tool[InputType]) ToolDescription() string {
	return t.Description
}

// Calculates the input schema from the generic InputType using reflection
func (t *Tool[InputType]) InputSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	var x InputType

	return reflector.Reflect(x)
}

// Calls the underlying function, unmarshalling the JSON first then calling the
// function with the native type
func (t *Tool[InputType]) Call(ctx context.Context, jsonInput []byte) (string, error) {
	var inputs InputType

	err := json.Unmarshal(jsonInput, &inputs)
	if err != nil {
		return "", err
	}

	return t.Func(ctx, inputs)
}
