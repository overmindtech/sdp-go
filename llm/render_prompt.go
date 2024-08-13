package llm

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"text/template"
	"time"

	"github.com/akedrou/textdiff"
	"github.com/akedrou/textdiff/myers"
	"github.com/overmindtech/sdp-go"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v2"
)

// Renders a prompt template, with access to a set of shared useful functions.
// These functions are:
//
// trim: Trims a string at most to the given length, adding ... in the middle
//
// itemToYAML: Converts an item to a YAML representation that is simplified and
// suitable for the LLM
//
// itemDiffToYAMLDiff: Converts an itemDiff to a YAML diff that is easier to
// read
//
// globallyUniqueName: Returns the globally unique name of an item or reference
//
// prettyStatus: Returns a human-readable status for an sdp.ItemDiffStatus
//
// prettySeverity: Returns a human-readable severity for an sdp.Risk_Severity
//
// prettyTime: Formats a time.Time or *timestamppb.Timestamp as a human-readable
// string
func RenderPrompt(tpl string, args any) (string, error) {
	parsed, err := template.New("tpl").Funcs(
		template.FuncMap{
			// TODO: Audit these and make sure they are used
			"trim":               trim,
			"itemToYAML":         itemToYAML,
			"itemDiffToYAMLDiff": itemDiffToYAMLDiff,
			"globallyUniqueName": globallyUniqueName,
			"prettyStatus":       prettyStatus,
			"prettySeverity":     prettySeverity,
			"prettyTime":         prettyTime,
		},
	).Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("could not parse template: %w", err)
	}
	var buf bytes.Buffer
	err = parsed.Execute(&buf, args)
	if err != nil {
		return "", fmt.Errorf("could not execute template: %w", err)
	}
	return buf.String(), nil
}

// Formats a time.Time or *timestamppb.Timestamp as a human-readable string
func prettyTime(input any) string {
	switch val := input.(type) {
	case time.Time:
		return val.Format(time.RFC3339)
	case *timestamppb.Timestamp:
		return val.AsTime().Format(time.RFC3339)
	default:
		return fmt.Sprint(input)
	}
}

// prettySeverity returns a human-readable string for the given severity
func prettySeverity(severity sdp.Risk_Severity) string {
	switch severity {
	case sdp.Risk_SEVERITY_UNSPECIFIED:
		return "unspecified"
	case sdp.Risk_SEVERITY_LOW:
		return "low"
	case sdp.Risk_SEVERITY_MEDIUM:
		return "medium"
	case sdp.Risk_SEVERITY_HIGH:
		return "high"
	default:
		return "unknown"
	}
}

type hasGloballyUniqueName interface {
	GloballyUniqueName() string
}

// Returns the globally unique name of an item, or an empty string if the item
// is nil
func globallyUniqueName(thing hasGloballyUniqueName) string {
	if thing == nil {
		return ""
	}
	return thing.GloballyUniqueName()
}

// prettyStatus returns a human-readable string for the given status
func prettyStatus(status any) string {
	switch status := status.(type) {
	case sdp.ItemDiffStatus:
		switch status {
		case sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNSPECIFIED:
			return "unspecified"
		case sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UNCHANGED:
			return "unchanged"
		case sdp.ItemDiffStatus_ITEM_DIFF_STATUS_CREATED:
			return "created"
		case sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED:
			return "updated"
		case sdp.ItemDiffStatus_ITEM_DIFF_STATUS_DELETED:
			return "deleted"
		case sdp.ItemDiffStatus_ITEM_DIFF_STATUS_REPLACED:
			return "replaced"
		default:
			return "unknown"
		}
	case sdp.ChangeStatus:
		switch status {
		case sdp.ChangeStatus_CHANGE_STATUS_UNSPECIFIED:
			return "unspecified"
		case sdp.ChangeStatus_CHANGE_STATUS_DEFINING:
			return "defining"
		case sdp.ChangeStatus_CHANGE_STATUS_HAPPENING:
			return "happening"
		case sdp.ChangeStatus_CHANGE_STATUS_PROCESSING:
			return "processing"
		case sdp.ChangeStatus_CHANGE_STATUS_DONE:
			return "done"
		default:
			return "unknown"
		}
	}

	return "unknown"
}

// trim trims a string to at most m length, adding ... in the middle
func trim(m int, s string) string {
	l := len(s)
	if l < m {
		// string is fine
		return s
	}

	overhang := l - m
	if overhang <= 3 {
		overhang = 4
	}
	toTrim := (l - overhang) / 2
	return s[:toTrim] + "\n…\n" + s[l-toTrim:]
}

// Converts an item to a YAML representation that is simplified and suitable for
// the LLM
func itemToYAML(item *sdp.Item) (string, error) {
	if item == nil {
		// If the item is nil, we can't convert it to YAML, this is likely when
		// the item is being created for example, the before state will be
		// empty. In this case we want to just return an empty string rather
		// than an error
		return "", nil
	}

	// We want to convert each item into the most helpful format we can for the
	// LLM, we don't want anything too SDP-specific in here, it should just be
	// easily understood for the layman/LLM
	itemForYaml := struct {
		Type               string         `yaml:"type"`
		GloballyUniqueName string         `yaml:"id"`
		Attributes         map[string]any `yaml:"attributes"`
		Health             string         `yaml:"health,omitempty"`

		// A slice of globally unique names
		RelatedItems []string `yaml:"related_items,omitempty"`
	}{
		Type:               item.GetType(),
		GloballyUniqueName: item.GloballyUniqueName(),
		Attributes:         cleanAsMap(item.GetAttributes().GetAttrStruct()),
	}

	// Only include health if it's known
	if health := item.GetHealth(); health != sdp.Health_HEALTH_UNKNOWN {
		itemForYaml.Health = health.String()
	}

	relatedGloballyUniqueNames := make([]string, len(item.GetLinkedItems()))

	for i, linkedItem := range item.GetLinkedItems() {
		relatedGloballyUniqueNames[i] = linkedItem.GetItem().GloballyUniqueName()
	}

	itemForYaml.RelatedItems = relatedGloballyUniqueNames

	yamlBytes, err := yaml.Marshal(itemForYaml)
	if err != nil {
		return "", err
	}

	return string(yamlBytes), nil
}

// Converts an itemDiff to a diff of the YAML format
func itemDiffToYAMLDiff(itemDiff *sdp.ItemDiff) (string, error) {
	if itemDiff == nil {
		return "", fmt.Errorf("itemDiff is nil")
	}

	// Convert the before and after items to YAML
	beforeYAML, err := itemToYAML(itemDiff.GetBefore())
	if err != nil {
		return "", err
	}

	afterYAML, err := itemToYAML(itemDiff.GetAfter())
	if err != nil {
		return "", err
	}

	// Diff the YAML
	edits := myers.ComputeEdits(beforeYAML, afterYAML)
	unified, err := textdiff.ToUnified("current", "proposed", beforeYAML, edits)

	if err != nil {
		return "", err
	}

	return unified, nil
}

// cleanAsMap takes a pointer to a structpb.Struct and returns a map[string]interface{}.
// It cleans the input struct by converting it into a map, where each field is represented
// by a key-value pair. The function handles different types of values in the struct,
// such as numbers, strings, booleans, nested structs, and lists.
// If a field's value is NaN or infinity, it is converted to a string representation.
// If a field's value is an empty string, it is removed from the map.
// If a field's value is an empty struct or list, it is removed from the map.
// The cleaned map is then returned as the result.
func cleanAsMap(x *structpb.Struct) map[string]any {
	f := x.GetFields()
	vs := make(map[string]interface{}, len(f))
	for k, v := range f {
		vs[k] = v.AsInterface()

		switch inner := v.GetKind().(type) {
		case *structpb.Value_NumberValue:
			if inner != nil {
				switch {
				case math.IsNaN(inner.NumberValue):
					vs[k] = "NaN"
				case math.IsInf(inner.NumberValue, +1):
					vs[k] = "Infinity"
				case math.IsInf(inner.NumberValue, -1):
					vs[k] = "-Infinity"
				default:
					vs[k] = inner.NumberValue
				}
			}
		case *structpb.Value_StringValue:
			if inner != nil {
				if inner.StringValue != "" {
					vs[k] = inner.StringValue
				} else {
					delete(vs, k)
				}
			}
		case *structpb.Value_BoolValue:
			if inner != nil {
				vs[k] = inner.BoolValue
			}
		case *structpb.Value_StructValue:
			if inner != nil {
				m := cleanAsMap(inner.StructValue)
				if len(m) > 0 {
					vs[k] = m
				} else {
					delete(vs, k)
				}
			}
		case *structpb.Value_ListValue:
			if inner != nil {
				l := cleanAsSlice(inner.ListValue)
				if len(l) > 0 {
					vs[k] = l
				} else {
					delete(vs, k)
				}
			}
		}
	}
	return vs
}

// AsSlice converts x to a general-purpose Go slice.
// The slice elements are converted by calling Value.AsInterface.
func cleanAsSlice(x *structpb.ListValue) []interface{} {
	vals := x.GetValues()
	vs := make([]interface{}, len(vals))
	for i, v := range vals {
		switch inner := v.GetKind().(type) {
		case *structpb.Value_NumberValue:
			if inner != nil {
				switch {
				case math.IsNaN(inner.NumberValue):
					vs[i] = "NaN"
				case math.IsInf(inner.NumberValue, +1):
					vs[i] = "Infinity"
				case math.IsInf(inner.NumberValue, -1):
					vs[i] = "-Infinity"
				default:
					vs[i] = inner.NumberValue
				}
			}
		case *structpb.Value_StringValue:
			if inner != nil && inner.StringValue != "" {
				vs[i] = inner.StringValue
			}
		case *structpb.Value_BoolValue:
			if inner != nil {
				vs[i] = inner.BoolValue
			}
		case *structpb.Value_StructValue:
			if inner != nil {
				vs[i] = cleanAsMap(inner.StructValue)
			}
		case *structpb.Value_ListValue:
			if inner != nil {
				vs[i] = cleanAsSlice(inner.ListValue)
			}
		}
	}
	return vs
}
