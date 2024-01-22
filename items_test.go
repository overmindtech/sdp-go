package sdp

import (
	"bytes"
	"context"
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
			"named struct": CustomStruct{
				Foo:  "foo",
				Bar:  "bar",
				Baz:  "baz",
				Time: time.Now(),
			},
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

			// In order to compare these reliably I'm going to do the following:
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

func TestDefaultTransformMap(t *testing.T) {
	input := map[string]interface{}{
		// Use a duration
		"hour": 1 * time.Hour,
	}

	attributes, err := ToAttributes(input)

	if err != nil {
		t.Fatal(err)
	}

	hour, err := attributes.Get("hour")

	if err != nil {
		t.Fatal(err)
	}

	if hour != "1h0m0s" {
		t.Errorf("Expected hour to be 1h0m0s, got %v", hour)
	}
}

func TestCustomTransforms(t *testing.T) {
	t.Run("redaction", func(t *testing.T) {
		type Secret struct {
			Value string
		}

		data := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "Hunter",
				"password": Secret{
					Value: "hunter2",
				},
			},
		}

		attributes, err := ToAttributesCustom(data, true, TransformMap{
			reflect.TypeOf(Secret{}): func(i interface{}) interface{} {
				// Remove it
				return nil
			},
		})

		if err != nil {
			t.Fatal(err)
		}

		user, err := attributes.Get("user")

		if err != nil {
			t.Fatal(err)
		}

		userMap, ok := user.(map[string]interface{})

		if !ok {
			t.Fatalf("Expected user to be a map, got %T", user)
		}

		if _, ok := userMap["password"]; ok {
			t.Error("Expected password to be removed")
		}
	})

	t.Run("map response", func(t *testing.T) {
		type Something struct {
			Foo string
			Bar string
		}

		data := map[string]interface{}{
			"something": Something{
				Foo: "foo",
				Bar: "bar",
			},
		}

		attributes, err := ToAttributesCustom(data, true, TransformMap{
			reflect.TypeOf(Something{}): func(i interface{}) interface{} {
				something := i.(Something)

				return map[string]string{
					"foo": something.Foo,
					"bar": something.Bar,
				}
			},
		})

		if err != nil {
			t.Fatal(err)
		}

		something, err := attributes.Get("something")

		if err != nil {
			t.Fatal(err)
		}

		somethingMap, ok := something.(map[string]interface{})

		if !ok {
			t.Fatalf("Expected something to be a map, got %T", something)
		}

		if somethingMap["foo"] != "foo" {
			t.Errorf("Expected foo to be foo, got %v", somethingMap["foo"])
		}

		if somethingMap["bar"] != "bar" {
			t.Errorf("Expected bar to be bar, got %v", somethingMap["bar"])
		}
	})
	t.Run("returns nil", func(t *testing.T) {
		type Something struct {
			Foo string
			Bar string
		}

		data := map[string]interface{}{
			"something": Something{
				Foo: "foo",
				Bar: "bar",
			},
			"else": nil,
		}

		attributes, err := ToAttributesCustom(data, true, TransformMap{
			reflect.TypeOf(Something{}): func(i interface{}) interface{} {
				return nil
			},
		})

		if err != nil {
			t.Fatal(err)
		}

		t.Log(attributes)
	})
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
			Scope:           "test",
			Attributes:      exampleAttributes,
			LinkedItemQueries: []*LinkedItemQuery{
				{
					Query: &Query{
						Type:   "user",
						Method: QueryMethod_GET,
						Query:  "Mike",
					},
				},
			},
			LinkedItems: []*LinkedItem{},
			Metadata: &Metadata{
				SourceName: "test",
				SourceQuery: &Query{
					Type:   "user",
					Method: QueryMethod_GET,
					Query:  "Dylan",
					Scope:  "testScope",
					UUID:   u[:],
				},
				Timestamp:             timestamppb.Now(),
				SourceDuration:        durationpb.New(100 * time.Millisecond),
				SourceDurationPerItem: durationpb.New(10 * time.Millisecond),
			},
			Health: Health_HEALTH_ERROR.Enum(),
			Tags: map[string]string{
				"foo": "bar",
			},
		}

		itemB := Item{}

		t.Run("Copying an item", func(t *testing.T) {
			itemA.Copy(&itemB)

			AssertItemsEqual(&itemA, &itemB, t)
		})
	})

	t.Run("With a party-filled item", func(t *testing.T) {
		itemA := Item{
			Type:            "user",
			UniqueAttribute: "name",
			Scope:           "test",
			Attributes:      exampleAttributes,
			LinkedItemQueries: []*LinkedItemQuery{
				{
					Query: &Query{
						Type:   "user",
						Method: QueryMethod_GET,
						Query:  "Mike",
					},
				},
			},
			LinkedItems: []*LinkedItem{},
			Metadata: &Metadata{
				Hidden:                true,
				SourceName:            "test",
				Timestamp:             timestamppb.Now(),
				SourceDuration:        durationpb.New(100 * time.Millisecond),
				SourceDurationPerItem: durationpb.New(10 * time.Millisecond),
			},
		}

		itemB := Item{}

		t.Run("Copying an item", func(t *testing.T) {
			itemA.Copy(&itemB)

			AssertItemsEqual(&itemA, &itemB, t)
		})
	})

	t.Run("With a minimal item", func(t *testing.T) {
		itemA := Item{
			Type:              "user",
			UniqueAttribute:   "name",
			Scope:             "test",
			Attributes:        exampleAttributes,
			LinkedItemQueries: []*LinkedItemQuery{},
			LinkedItems:       []*LinkedItem{},
		}

		itemB := Item{}

		t.Run("Copying an item", func(t *testing.T) {
			itemA.Copy(&itemB)

			AssertItemsEqual(&itemA, &itemB, t)
		})
	})

}

func AssertItemsEqual(itemA *Item, itemB *Item, t *testing.T) {
	if itemA.Scope != itemB.Scope {
		t.Error("Scope did not match")
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

	if len(itemA.LinkedItemQueries) != len(itemB.LinkedItemQueries) {
		t.Error("LinkedItemQueries length did not match")
	}

	if len(itemA.LinkedItemQueries) > 0 {
		if itemA.LinkedItemQueries[0].Query.Type != itemB.LinkedItemQueries[0].Query.Type {
			t.Error("LinkedItemQueries[0].Type did not match")
		}
	}

	if len(itemA.LinkedItems) != len(itemB.LinkedItems) {
		t.Error("LinkedItems length did not match")
	}

	if len(itemA.LinkedItems) > 0 {
		if itemA.LinkedItems[0].Item.Type != itemB.LinkedItems[0].Item.Type {
			t.Error("LinkedItemQueries[0].Type did not match")
		}
	}

	for k, v := range itemA.Tags {
		if itemB.Tags[k] != v {
			t.Errorf("Tags[%v] did not match", k)
		}
	}

	if itemA.Health == nil {
		if itemB.Health != nil {
			t.Errorf("mismatched health nil and %v", *itemB.Health)
		}
	} else {
		if itemB.Health == nil {
			t.Errorf("mismatched health %v and nil", *itemA.Health)

		} else {
			if *itemA.Health != *itemB.Health {
				t.Errorf("mismatched health %v and %v", *itemA.Health, *itemB.Health)
			}
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

		if itemA.Metadata.Hidden != itemB.Metadata.Hidden {
			t.Error("Metadata.Hidden does not match")
		}

		if itemA.Metadata.SourceQuery != nil {
			if itemA.Metadata.SourceQuery.Scope != itemB.Metadata.SourceQuery.Scope {
				t.Error("Metadata.SourceQuery.Scope does not match")
			}

			if itemA.Metadata.SourceQuery.Method != itemB.Metadata.SourceQuery.Method {
				t.Error("Metadata.SourceQuery.Method does not match")
			}

			if itemA.Metadata.SourceQuery.Query != itemB.Metadata.SourceQuery.Query {
				t.Error("Metadata.SourceQuery.Query does not match")
			}

			if itemA.Metadata.SourceQuery.Type != itemB.Metadata.SourceQuery.Type {
				t.Error("Metadata.SourceQuery.Type does not match")
			}

			if !bytes.Equal(itemA.Metadata.SourceQuery.UUID, itemB.Metadata.SourceQuery.UUID) {
				t.Error("Metadata.SourceQuery.UUID does not match")
			}
		}
	}
}

func TestTimeoutContext(t *testing.T) {
	q := Query{
		Type:   "person",
		Method: QueryMethod_GET,
		Query:  "foo",
		RecursionBehaviour: &Query_RecursionBehaviour{
			LinkDepth: 2,
		},
		IgnoreCache: false,
		Deadline:    timestamppb.New(time.Now().Add(10 * time.Millisecond)),
	}

	ctx, cancel := q.TimeoutContext(context.Background())
	defer cancel()

	select {
	case <-time.After(20 * time.Millisecond):
		t.Error("Context did not time out after 10ms")
	case <-ctx.Done():
		// This is good
	}
}

func TestToAttributesViaJson(t *testing.T) {
	// Create a random struct
	test1 := struct {
		Foo  string
		Bar  bool
		Blip []string
		Baz  struct {
			Zap string
			Bam int
		}
	}{
		Foo: "foo",
		Bar: false,
		Blip: []string{
			"yes",
			"I",
			"blip",
		},
		Baz: struct {
			Zap string
			Bam int
		}{
			Zap: "negative",
			Bam: 42,
		},
	}

	attributes, err := ToAttributesViaJson(test1)

	if err != nil {
		t.Fatal(err)
	}

	if foo, err := attributes.Get("Foo"); err != nil || foo != "foo" {
		t.Errorf("Expected Foo to be 'foo', got %v, err: %v", foo, err)
	}
}

func TestAttributesGet(t *testing.T) {
	mapData := map[string]interface{}{
		"foo": "bar",
		"nest": map[string]interface{}{
			"nest2": map[string]string{
				"nest3": "nestValue",
			},
		},
	}

	attr, err := ToAttributes(mapData)

	if err != nil {
		t.Fatal(err)
	}

	if v, err := attr.Get("foo"); err != nil || v != "bar" {
		t.Errorf("expected Get(\"foo\") to be bar, got %v", v)
	}

	if v, err := attr.Get("nest.nest2.nest3"); err != nil || v != "nestValue" {
		t.Errorf("expected Get(\"nest.nest2.nest3\") to be nestValue, got %v", v)
	}
}

func TestAttributesSet(t *testing.T) {
	mapData := map[string]interface{}{
		"foo": "bar",
		"nest": map[string]interface{}{
			"nest2": map[string]string{
				"nest3": "nestValue",
			},
		},
	}

	attr, err := ToAttributes(mapData)

	if err != nil {
		t.Fatal(err)
	}

	err = attr.Set("foo", "baz")

	if err != nil {
		t.Error(err)
	}

	if v, err := attr.Get("foo"); err != nil || v != "baz" {
		t.Errorf("expected Get(\"foo\") to be baz, got %v", v)
	}
}
