package sdp

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ToAttributesTest struct {
	Name  string
	Input map[string]interface{}
}

type CustomString string

var Dylan CustomString = "Dylan"

type CustomBool bool

var Bool1 CustomBool = false
var NilPointerBool *bool

type CustomStruct struct {
	Foo      string        `json:",omitempty"`
	Bar      string        `json:",omitempty"`
	Baz      string        `json:",omitempty"`
	Time     time.Time     `json:",omitempty"`
	Duration time.Duration `json:",omitempty"`
}

var Struct CustomStruct = CustomStruct{
	Foo:      "foo",
	Bar:      "bar",
	Baz:      "baz",
	Time:     time.Now(),
	Duration: (10 * time.Minute),
}

var ToAttributesTests = []ToAttributesTest{
	{
		Name: "Basic strings map",
		Input: map[string]interface{}{
			"firstName": "Dylan",
			"lastName":  "Ratcliffe",
		},
	},
	{
		Name: "Arrays map",
		Input: map[string]interface{}{
			"empty": []string{},
			"single-level": []string{
				"one",
				"two",
			},
			"multi-level": [][]string{
				{
					"one-one",
					"one-two",
				},
				{
					"two-one",
					"two-two",
				},
			},
		},
	},
	{
		Name: "Nested strings maps",
		Input: map[string]interface{}{
			"strings map": map[string]string{
				"foo": "bar",
			},
		},
	},
	{
		Name: "Nested integer map",
		Input: map[string]interface{}{
			"numbers map": map[string]int{
				"one": 1,
				"two": 2,
			},
		},
	},
	{
		Name: "Nested string-array map",
		Input: map[string]interface{}{
			"arrays map": map[string][]string{
				"dogs": {
					"pug",
					"also pug",
				},
			},
		},
	},
	{
		Name: "Nested non-string keys map",
		Input: map[string]interface{}{
			"non-string keys": map[int]string{
				1: "one",
				2: "two",
				3: "three",
			},
		},
	},
	{
		Name: "Composite types",
		Input: map[string]interface{}{
			"underlying string": Dylan,
			"underlying bool":   Bool1,
		},
	},
	{
		Name: "Pointers",
		Input: map[string]interface{}{
			"pointer bool":    &Bool1,
			"pointer string":  &Dylan,
			"pointer to zero": NilPointerBool,
		},
	},
	{
		Name: "structs",
		Input: map[string]interface{}{
			"named struct": Struct,
			"anon struct": struct {
				Yes bool
			}{
				Yes: true,
			},
		},
	},
	{
		Name: "Zero-value structs",
		Input: map[string]interface{}{
			"something": CustomStruct{
				Foo:  "yes",
				Time: time.Now(),
			},
		},
	},
}

func TestToAttributes(t *testing.T) {
	for _, tat := range ToAttributesTests {
		t.Run(tat.Name, func(t *testing.T) {
			var inputBytes []byte
			var attributesBytes []byte
			var inputJSON string
			var attributesJSON string
			var attributes *ItemAttributes
			var err error

			// Convert the input to Attributes
			attributes, err = ToAttributes(tat.Input)

			if err != nil {
				t.Fatal(err)
			}

			// In order to compate these reliably I'm going to do the following:
			//
			// 1. Convert to JSON
			// 2. Convert back again
			// 3. Compare with reflect.DeepEqual()

			// Convert the input to JSON
			inputBytes, err = json.MarshalIndent(tat.Input, "", "  ")

			if err != nil {
				t.Fatal(err)
			}

			// Convert the attributes to JSON
			attributesBytes, err = json.MarshalIndent(attributes.AttrStruct.AsMap(), "", "  ")

			if err != nil {
				t.Fatal(err)
			}

			var input map[string]interface{}
			var output map[string]interface{}

			err = json.Unmarshal(inputBytes, &input)

			if err != nil {
				t.Fatal(err)
			}

			err = json.Unmarshal(attributesBytes, &output)

			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(input, output) {
				// Convert to strings for printing
				inputJSON = string(inputBytes)
				attributesJSON = string(attributesBytes)

				t.Errorf("JSON did not match (note that order of map keys doesn't matter)\nInput: %v\nAttributes: %v", inputJSON, attributesJSON)
			}
		})

	}
}

func TestCopy(t *testing.T) {
	exampleAttributes, err := ToAttributes(map[string]interface{}{
		"name":   "Dylan",
		"friend": "Mike",
		"age":    27,
	})

	if err != nil {
		t.Fatalf("Could not convert to attributes: %v", err)
	}

	t.Run("With a complete item", func(t *testing.T) {
		u := uuid.New()

		itemA := Item{
			Type:            "user",
			UniqueAttribute: "name",
			Context:         "test",
			Attributes:      exampleAttributes,
			LinkedItemRequests: []*ItemRequest{
				{
					Type:   "user",
					Method: RequestMethod_GET,
					Query:  "Mike",
				},
			},
			LinkedItems: []*Reference{},
			Metadata: &Metadata{
				SourceName: "test",
				SourceRequest: &ItemRequest{
					Type:    "user",
					Method:  RequestMethod_GET,
					Query:   "Dylan",
					Context: "testContext",
					UUID:    u[:],
				},
				Timestamp:             timestamppb.Now(),
				SourceDuration:        durationpb.New(100 * time.Millisecond),
				SourceDurationPerItem: durationpb.New(10 * time.Millisecond),
			},
		}

		itemB := Item{}

		t.Run("Copying an item", func(t *testing.T) {
			itemA.Copy(&itemB)

			CompareItems(&itemA, &itemB, t)
		})
	})

	t.Run("With a party-filled item", func(t *testing.T) {
		itemA := Item{
			Type:            "user",
			UniqueAttribute: "name",
			Context:         "test",
			Attributes:      exampleAttributes,
			LinkedItemRequests: []*ItemRequest{
				{
					Type:   "user",
					Method: RequestMethod_GET,
					Query:  "Mike",
				},
			},
			LinkedItems: []*Reference{},
			Metadata: &Metadata{
				SourceName:            "test",
				Timestamp:             timestamppb.Now(),
				SourceDuration:        durationpb.New(100 * time.Millisecond),
				SourceDurationPerItem: durationpb.New(10 * time.Millisecond),
			},
		}

		itemB := Item{}

		t.Run("Copying an item", func(t *testing.T) {
			itemA.Copy(&itemB)

			CompareItems(&itemA, &itemB, t)
		})
	})

	t.Run("With a minimal item", func(t *testing.T) {
		itemA := Item{
			Type:               "user",
			UniqueAttribute:    "name",
			Context:            "test",
			Attributes:         exampleAttributes,
			LinkedItemRequests: []*ItemRequest{},
			LinkedItems:        []*Reference{},
		}

		itemB := Item{}

		t.Run("Copying an item", func(t *testing.T) {
			itemA.Copy(&itemB)

			CompareItems(&itemA, &itemB, t)
		})
	})

}

func CompareItems(itemA *Item, itemB *Item, t *testing.T) {
	if itemA.Context != itemB.Context {
		t.Error("Context did not match")
	}

	if itemA.Type != itemB.Type {
		t.Error("Type did not match")
	}

	if itemA.UniqueAttribute != itemB.UniqueAttribute {
		t.Error("UniqueAttribute did not match")
	}

	var nameA interface{}
	var nameB interface{}
	var err error

	nameA, err = itemA.Attributes.Get("name")

	if err != nil {
		t.Error(err)
	}

	nameB, err = itemB.Attributes.Get("name")

	if err != nil {
		t.Error(err)
	}

	if nameA != nameB {
		t.Error("Attributes.nam did not match")

	}

	if len(itemA.LinkedItemRequests) != len(itemB.LinkedItemRequests) {
		t.Error("LinkedItemRequests length did not match")
	}

	if len(itemA.LinkedItemRequests) > 0 {
		if itemA.LinkedItemRequests[0].Type != itemB.LinkedItemRequests[0].Type {
			t.Error("LinkedItemRequests[0].Type did not match")
		}
	}

	if len(itemA.LinkedItems) != len(itemB.LinkedItems) {
		t.Error("LinkedItems length did not match")
	}

	if len(itemA.LinkedItems) > 0 {
		if itemA.LinkedItems[0].Type != itemB.LinkedItems[0].Type {
			t.Error("LinkedItemRequests[0].Type did not match")
		}
	}

	if itemA.Metadata != nil {
		if itemA.Metadata.SourceDuration.String() != itemB.Metadata.SourceDuration.String() {
			t.Error("SourceDuration did not match")
		}

		if itemA.Metadata.SourceDurationPerItem.String() != itemB.Metadata.SourceDurationPerItem.String() {
			t.Error("SourceDurationPerItem did not match")
		}

		if itemA.Metadata.SourceName != itemB.Metadata.SourceName {
			t.Error("SourceName did not match")
		}

		if itemA.Metadata.Timestamp.String() != itemB.Metadata.Timestamp.String() {
			t.Error("Timestamp did not match")
		}

		if itemA.Metadata.SourceRequest != nil {
			if itemA.Metadata.SourceRequest.Context != itemB.Metadata.SourceRequest.Context {
				t.Error("Metadata.SourceRequest.Context does not match")
			}

			if itemA.Metadata.SourceRequest.Method != itemB.Metadata.SourceRequest.Method {
				t.Error("Metadata.SourceRequest.Method does not match")
			}

			if itemA.Metadata.SourceRequest.Query != itemB.Metadata.SourceRequest.Query {
				t.Error("Metadata.SourceRequest.Query does not match")
			}

			if itemA.Metadata.SourceRequest.Type != itemB.Metadata.SourceRequest.Type {
				t.Error("Metadata.SourceRequest.Type does not match")
			}

			uuidA, _ := uuid.FromBytes(itemA.Metadata.SourceRequest.UUID)
			uuidB, _ := uuid.FromBytes(itemB.Metadata.SourceRequest.UUID)

			if uuidA.String() != uuidB.String() {
				t.Error("Metadata.SourceRequest.UUID does not match")
			}
		}
	}
}

func TestTimeoutContext(t *testing.T) {
	r := ItemRequest{
		Type:        "person",
		Method:      RequestMethod_GET,
		Query:       "foo",
		LinkDepth:   2,
		IgnoreCache: false,
		Timeout:     durationpb.New(10 * time.Millisecond),
	}

	ctx, cancel := r.TimeoutContext()
	defer cancel()

	select {
	case <-time.After(20 * time.Millisecond):
		t.Error("Context did not time out after 10ms")
	case <-ctx.Done():
		// This is good
	}
}
