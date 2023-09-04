package sdp

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WILDCARD = "*"

// Copy copies all information from one item pointer to another
func (liq *BlastPropagation) Copy(dest *BlastPropagation) {
	dest.In = liq.GetIn()
	dest.Out = liq.GetOut()
}

// Copy copies all information from one item pointer to another
func (liq *LinkedItemQuery) Copy(dest *LinkedItemQuery) {
	dest.Query = &Query{}
	if liq.Query != nil {
		liq.Query.Copy(dest.Query)
	}
	dest.BlastPropagation = &BlastPropagation{}
	if liq.BlastPropagation != nil {
		liq.BlastPropagation.Copy(dest.BlastPropagation)
	}
}

// Copy copies all information from one item pointer to another
func (li *LinkedItem) Copy(dest *LinkedItem) {
	dest.Item = &Reference{}
	if li.Item != nil {
		li.Item.Copy(dest.Item)
	}
	dest.BlastPropagation = &BlastPropagation{}
	if li.BlastPropagation != nil {
		li.BlastPropagation.Copy(dest.BlastPropagation)
	}
}

// UniqueAttributeValue returns the value of whatever the Unique Attribute is
// for this item. This will then be converted to a string and returned
func (i *Item) UniqueAttributeValue() string {
	var value interface{}
	var err error

	value, err = i.GetAttributes().Get(i.UniqueAttribute)

	if err == nil {
		return fmt.Sprint(value)
	}

	return ""
}

// Reference returns an SDP reference for the item
func (i *Item) Reference() *Reference {
	return &Reference{
		Scope:                i.Scope,
		Type:                 i.Type,
		UniqueAttributeValue: i.UniqueAttributeValue(),
	}
}

// GloballyUniqueName Returns a string that defines the Item globally. This a
// combination of the following values:
//
//   - scope
//   - type
//   - uniqueAttributeValue
//
// They are concatenated with dots (.)
func (i *Item) GloballyUniqueName() string {
	return strings.Join([]string{
		i.GetScope(),
		i.GetType(),
		i.UniqueAttributeValue(),
	},
		".",
	)
}

// Copy copies all information from one item pointer to another
func (i *Item) Copy(dest *Item) {
	// Values can be copied directly
	dest.Type = i.Type
	dest.UniqueAttribute = i.UniqueAttribute
	dest.Scope = i.Scope

	// We need to check that any pointers are actually populated with pointers
	// to somewhere in memory. If they are nil then there is no data structure
	// to copy the data into, therefore we need to create it first
	if dest.Metadata == nil {
		dest.Metadata = &Metadata{}
	}

	if dest.Attributes == nil {
		dest.Attributes = &ItemAttributes{}
	}

	i.Metadata.Copy(dest.Metadata)
	i.Attributes.Copy(dest.Attributes)

	dest.LinkedItemQueries = make([]*LinkedItemQuery, 0)
	dest.LinkedItems = make([]*LinkedItem, 0)

	for _, r := range i.LinkedItemQueries {
		liq := &LinkedItemQuery{}

		r.Copy(liq)

		dest.LinkedItemQueries = append(dest.LinkedItemQueries, liq)
	}

	for _, r := range i.LinkedItems {
		newLinkedItem := &LinkedItem{}

		r.Copy(newLinkedItem)

		dest.LinkedItems = append(dest.LinkedItems, newLinkedItem)
	}

	if i.Health == nil {
		dest.Health = nil
	} else {
		dest.Health = i.Health.Enum()
	}
}

// Hash Returns a 12 character hash for the item. This is likely but not
// guaranteed to be unique. The hash is calculated using the GloballyUniqueName
func (i *Item) Hash() string {
	return hashSum(([]byte(fmt.Sprint(i.GloballyUniqueName()))))
}

// Hash Returns a 12 character hash for the item. This is likely but not
// guaranteed to be unique. The hash is calculated using the GloballyUniqueName
func (r *Reference) Hash() string {
	return hashSum(([]byte(fmt.Sprint(r.GloballyUniqueName()))))
}

// GloballyUniqueName Returns a string that defines the Item globally. This a
// combination of the following values:
//
//   - scope
//   - type
//   - uniqueAttributeValue
//
// They are concatenated with dots (.)
func (r *Reference) GloballyUniqueName() string {
	return strings.Join([]string{
		r.GetScope(),
		r.GetType(),
		r.GetUniqueAttributeValue(),
	},
		".",
	)
}

// Copy copies all information from one Reference pointer to another
func (r *Reference) Copy(dest *Reference) {
	dest.Type = r.Type
	dest.UniqueAttributeValue = r.UniqueAttributeValue
	dest.Scope = r.Scope
}

// Copy copies all information from one Metadata pointer to another
func (m *Metadata) Copy(dest *Metadata) {
	if m == nil {
		// Protect from copy being called on a nil pointer
		return
	}

	dest.SourceName = m.SourceName
	dest.Hidden = m.Hidden

	if m.SourceQuery != nil {
		dest.SourceQuery = &Query{}
		m.SourceQuery.Copy(dest.SourceQuery)
	}

	dest.Timestamp = &timestamppb.Timestamp{
		Seconds: m.Timestamp.GetSeconds(),
		Nanos:   m.Timestamp.GetNanos(),
	}

	dest.SourceDuration = &durationpb.Duration{
		Seconds: m.SourceDuration.GetSeconds(),
		Nanos:   m.SourceDuration.GetNanos(),
	}

	dest.SourceDurationPerItem = &durationpb.Duration{
		Seconds: m.SourceDurationPerItem.GetSeconds(),
		Nanos:   m.SourceDurationPerItem.GetNanos(),
	}
}

// Copy copies all information from one CancelQuery pointer to another
func (c *CancelQuery) Copy(dest *CancelQuery) {
	if c == nil {
		return
	}

	dest.UUID = c.UUID
}

// Get Returns the value of a given attribute by name. If the attribute is
// a nested hash, nested values can be referenced using dot notation e.g.
// location.country
func (a *ItemAttributes) Get(name string) (interface{}, error) {
	var result interface{}

	// Start at the beginning of the map, we will then traverse down as required
	result = a.GetAttrStruct().AsMap()

	for _, section := range strings.Split(name, ".") {
		// Check that the data we're using is in the supported format
		var m map[string]interface{}

		m, isMap := result.(map[string]interface{})

		if !isMap {
			return nil, fmt.Errorf("attribute %v not found", name)
		}

		v, keyExists := m[section]

		if keyExists {
			result = v
		} else {
			return nil, fmt.Errorf("attribute %v not found", name)
		}
	}

	return result, nil
}

// Set sets an attribute. Values are converted to structpb versions and an error
// will be returned if this fails. Note that this does *not* yet support
// dot notation e.g. location.country
func (a *ItemAttributes) Set(name string, value interface{}) error {
	// Check to make sure that the pointer is not nil
	if a == nil {
		return errors.New("Set called on nil pointer")
	}

	// Ensure that this interface will be able to be converted to a struct value
	sanitizedValue := sanitizeInterface(value, false)
	structValue, err := structpb.NewValue(sanitizedValue)

	if err != nil {
		return err
	}

	fields := a.GetAttrStruct().GetFields()

	fields[name] = structValue

	return nil
}

// Copy copies all information from one ItemAttributes pointer to another
func (a *ItemAttributes) Copy(dest *ItemAttributes) {
	m := a.GetAttrStruct().AsMap()

	dest.AttrStruct, _ = structpb.NewStruct(m)
}

// Copy copies all information from one Query pointer to another
func (qrb *Query_RecursionBehaviour) Copy(dest *Query_RecursionBehaviour) {
	dest.LinkDepth = qrb.LinkDepth
	dest.FollowOnlyBlastPropagation = qrb.FollowOnlyBlastPropagation
}

// Subject returns a NATS subject for all traffic relating to this query
func (q *Query) Subject() string {
	return fmt.Sprintf("query.%v", q.ParseUuid())
}

// Copy copies all information from one Query pointer to another
func (q *Query) Copy(dest *Query) {
	dest.Type = q.Type
	dest.Method = q.Method
	dest.Query = q.Query
	dest.RecursionBehaviour = &Query_RecursionBehaviour{}
	if q.RecursionBehaviour != nil {
		q.RecursionBehaviour.Copy(dest.RecursionBehaviour)
	}
	dest.Scope = q.Scope
	dest.IgnoreCache = q.IgnoreCache
	dest.UUID = q.UUID

	if q.Timeout != nil {
		dest.Timeout = &durationpb.Duration{
			Seconds: q.Timeout.Seconds,
			Nanos:   q.Timeout.Nanos,
		}
	}
}

// TimeoutContext returns a context and cancel function representing the timeout
// for this request
func (q *Query) TimeoutContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if q == nil || q.Timeout == nil {
		return context.WithCancel(ctx)
	}

	// If the timeout is 0, treat that as infinite
	if q.Timeout.Nanos == 0 && q.Timeout.Seconds == 0 {
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(ctx, q.Timeout.AsDuration())
}

// ParseUuid returns this request's UUID. If there's an error parsing it,
// generates and stores a fresh one
func (r *Query) ParseUuid() uuid.UUID {
	// Extract and parse the UUID
	reqUUID, uuidErr := uuid.FromBytes(r.UUID)
	if uuidErr != nil {
		reqUUID = uuid.New()
		r.UUID = reqUUID[:]
	}
	return reqUUID
}

func (x *QueryError) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *CancelQuery) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *UndoQuery) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *Expand) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *UndoExpand) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

// Converts a map[string]interface{} to an ItemAttributes object, sorting all
// slices alphabetically.This should be used when the item doesn't contain array
// attributes that are explicitly sorted, especially if these are sometimes
// returned in a different order
func ToAttributesSorted(m map[string]interface{}) (*ItemAttributes, error) {
	return toAttributes(m, true)
}

// ToAttributes Converts a map[string]interface{} to an ItemAttributes object
func ToAttributes(m map[string]interface{}) (*ItemAttributes, error) {
	return toAttributes(m, false)
}

func toAttributes(m map[string]interface{}, sort bool) (*ItemAttributes, error) {
	if m == nil {
		return nil, nil
	}

	var s map[string]*structpb.Value
	var err error

	s = make(map[string]*structpb.Value)

	// Loop over the map
	for k, v := range m {
		if v == nil {
			// If the value is nil then ignore
			continue
		}

		sanitizedValue := sanitizeInterface(v, sort)
		structValue, err := structpb.NewValue(sanitizedValue)

		if err != nil {
			return nil, err
		}

		s[k] = structValue
	}

	return &ItemAttributes{
		AttrStruct: &structpb.Struct{
			Fields: s,
		},
	}, err
}

// ToAttributesViaJson Converts any struct to a set of attributes by marshalling
// to JSON and then back again. This is less performant than ToAttributes() but
// does save work when copying large structs to attributes in their entirety
func ToAttributesViaJson(v interface{}) (*ItemAttributes, error) {
	b, err := json.Marshal(v)

	if err != nil {
		return nil, err
	}

	var m map[string]interface{}

	err = json.Unmarshal(b, &m)

	if err != nil {
		return nil, err
	}

	return ToAttributes(m)
}

// sanitizeInterface Ensures that en interface is in a format that can be
// converted to a protobuf value. The structpb.ToValue() function expects things
// to be in one of the following formats:
//
//	╔════════════════════════╤════════════════════════════════════════════╗
//	║ Go type                │ Conversion                                 ║
//	╠════════════════════════╪════════════════════════════════════════════╣
//	║ nil                    │ stored as NullValue                        ║
//	║ bool                   │ stored as BoolValue                        ║
//	║ int, int32, int64      │ stored as NumberValue                      ║
//	║ uint, uint32, uint64   │ stored as NumberValue                      ║
//	║ float32, float64       │ stored as NumberValue                      ║
//	║ string                 │ stored as StringValue; must be valid UTF-8 ║
//	║ []byte                 │ stored as StringValue; base64-encoded      ║
//	║ map[string]interface{} │ stored as StructValue                      ║
//	║ []interface{}          │ stored as ListValue                        ║
//	╚════════════════════════╧════════════════════════════════════════════╝
//
// However this means that a data type like []string won't work, despite the
// function being perfectly able to represent it in a protobuf struct. This
// function does its best to example the available data type to ensure that as
// long as the data can in theory be represented by a protobuf struct, the
// conversion will work.
func sanitizeInterface(i interface{}, sortArrays bool) interface{} {
	v := reflect.ValueOf(i)
	t := v.Type()

	if i == nil {
		return nil
	}

	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int:
		return v.Int()
	case reflect.Int8:
		return v.Int()
	case reflect.Int16:
		return v.Int()
	case reflect.Int32:
		return v.Int()
	case reflect.Int64:
		return v.Int()
	case reflect.Uint:
		return v.Uint()
	case reflect.Uint8:
		return v.Uint()
	case reflect.Uint16:
		return v.Uint()
	case reflect.Uint32:
		return v.Uint()
	case reflect.Uint64:
		return v.Uint()
	case reflect.Float32:
		return v.Float()
	case reflect.Float64:
		return v.Float()
	case reflect.String:
		return fmt.Sprint(v)
	case reflect.Array, reflect.Slice:
		// We need to check the type of each element in the array and do
		// conversion on that

		// returnSlice Returns the array in the format that protobuf can deal with
		var returnSlice []interface{}

		returnSlice = make([]interface{}, v.Len())

		for index := 0; index < v.Len(); index++ {
			returnSlice[index] = sanitizeInterface(v.Index(index).Interface(), sortArrays)
		}

		if sortArrays {
			sortInterfaceArray(returnSlice)
		}

		return returnSlice
	case reflect.Map:
		var returnMap map[string]interface{}

		returnMap = make(map[string]interface{})

		for _, mapKey := range v.MapKeys() {
			// Convert the key to a string
			stringKey := fmt.Sprint(mapKey.Interface())

			// Convert the value to a compatible interface
			mapValueInterface := v.MapIndex(mapKey).Interface()
			zeroValueInterface := reflect.Zero(v.MapIndex(mapKey).Type()).Interface()

			// Only use the item if it isn't zero
			if !reflect.DeepEqual(mapValueInterface, zeroValueInterface) {
				value := sanitizeInterface(v.MapIndex(mapKey).Interface(), sortArrays)

				returnMap[stringKey] = value
			}
		}

		return returnMap
	case reflect.Struct:
		// Special Cases
		switch x := i.(type) {
		case time.Time:
			// If it's a time we just want to print in ISO8601
			return x.Format(time.RFC3339Nano)
		case time.Duration:
			// If it's duration we want to print in a parsable format
			return x.String()
		default:
			// In the case of a struct we basically want to turn it into a
			// map[string]interface{}
			var returnMap map[string]interface{}

			returnMap = make(map[string]interface{})

			// Range over fields
			n := t.NumField()
			for i := 0; i < n; i++ {
				field := t.Field(i)

				if field.PkgPath != "" {
					// If this has a PkgPath then it is an un-exported fiend and
					// should be ignored
					continue
				}

				// Get the zero value for this field
				zeroValue := reflect.Zero(field.Type).Interface()
				fieldValue := v.Field(i).Interface()

				// Check if the field is it's nil value
				// Check if there actually was a field with that name
				if !reflect.DeepEqual(fieldValue, zeroValue) {
					returnMap[field.Name] = fieldValue
				}
			}

			return sanitizeInterface(returnMap, sortArrays)
		}
	case reflect.Ptr:
		// Get the zero value for this field
		zero := reflect.Zero(t)

		// Check if the field is it's nil value
		if reflect.DeepEqual(v, zero) {
			return nil
		}

		return sanitizeInterface(v.Elem().Interface(), sortArrays)
	default:
		// If we don't recognize the type then we need to see what the
		// underlying type is and see if we can convert that
		return i
	}
}

// Sorts an interface slice by converting each item to a string and sorting
// these strings
func sortInterfaceArray(input []interface{}) {
	sort.Slice(input, func(i, j int) bool {
		return fmt.Sprint(input[i]) < fmt.Sprint(input[j])
	})
}

func hashSum(b []byte) string {
	var shaSum [20]byte
	var paddedEncoding *base32.Encoding
	var unpaddedEncoding *base32.Encoding

	shaSum = sha1.Sum(b)

	// We need to specify a custom encoding here since dGraph has fairly strict
	// requirements about what name a variable can have
	paddedEncoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyzABCDEF")

	// We also can't have padding since "=" is not allowed in variable names
	unpaddedEncoding = paddedEncoding.WithPadding(base32.NoPadding)

	return unpaddedEncoding.EncodeToString(shaSum[:11])
}
