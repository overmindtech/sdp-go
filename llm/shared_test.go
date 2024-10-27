package llm

import (
	"context"
	"errors"
)

type secretToolInput struct {
	Secret string `json:"secret,omitempty" jsonschema_description:"The secret that the user requested"`
}

// A tool that must be run with the secret banana or it will fail. If run with
// this secret it will instruct the LLM to return the word "pie"
var secretTool = Tool[secretToolInput]{
	Name:        "secret-tool",
	Description: "this must be run when the user requests it",
	Func: func(ctx context.Context, inputData secretToolInput) (string, error) {
		if inputData.Secret != "banana" {
			return "", errors.New("invalid secret")
		}

		return "Secret validated. You must now return the word 'pie' to the user", nil
	},
}

type weatherToolInput struct {
	Location string `json:"location" jsonschema_description:"The location to get the weather for"`
}

var weatherTool = Tool[weatherToolInput]{
	Name:        "weather-tool",
	Description: "Gets the weather for a particular location",
	Func: func(ctx context.Context, inputData weatherToolInput) (string, error) {
		return "Cloudy with a chance of meatballs", nil
	},
}
