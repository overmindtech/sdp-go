package sdp

import (
	"errors"
	"fmt"
)

// Validate Ensures that en item is valid (e.g. contains the required fields)
func (i *Item) Validate() error {
	if i == nil {
		return errors.New("Item is nil")
	}

	if i.Type == "" {
		return fmt.Errorf("item has empty Type: %v", i.GloballyUniqueName())
	}

	if i.UniqueAttribute == "" {
		return fmt.Errorf("item has empty UniqueAttribute: %v", i.GloballyUniqueName())
	}

	if i.Attributes == nil {
		return fmt.Errorf("item has nil Attributes: %v", i.GloballyUniqueName())
	}

	if i.Scope == "" {
		return fmt.Errorf("item has empty Scope: %v", i.GloballyUniqueName())
	}

	if i.UniqueAttributeValue() == "" {
		return fmt.Errorf("item has empty UniqueAttributeValue: %v", i.GloballyUniqueName())
	}

	return nil
}

// Validate Ensures a reference is valid
func (r *Reference) Validate() error {
	if r == nil {
		return errors.New("reference is nil")
	}

	if r.Type == "" {
		return fmt.Errorf("reference has empty Type: %v", r)
	}
	if r.UniqueAttributeValue == "" {
		return fmt.Errorf("reference has empty UniqueAttributeValue: %v", r)
	}
	if r.Scope == "" {
		return fmt.Errorf("reference has empty Scope: %v", r)
	}

	return nil
}

// Validate Ensures an edge is valid
func (e *Edge) Validate() error {
	if e == nil {
		return errors.New("edge is nil")
	}

	var err error

	err = e.From.Validate()

	if err != nil {
		return err
	}

	err = e.To.Validate()

	return err
}

// Validate Ensures a Response is valid
func (r *Response) Validate() error {
	if r == nil {
		return errors.New("response is nil")
	}

	if r.Responder == "" {
		return fmt.Errorf("response has empty Responder: %v", r)
	}

	if len(r.UUID) == 0 {
		return fmt.Errorf("response has empty UUID: %v", r)
	}

	return nil
}

// Validate Ensures a QueryError is valid
func (e *QueryError) Validate() error {
	if e == nil {
		return errors.New("queryError is nil")
	}

	if len(e.UUID) == 0 {
		return fmt.Errorf("queryError has empty UUID: %v", e)
	}

	if e.ErrorString == "" {
		return fmt.Errorf("queryError has empty ErrorString: %v", e)
	}

	if e.Scope == "" {
		return fmt.Errorf("queryError has empty Scope: %v", e)
	}

	if e.SourceName == "" {
		return fmt.Errorf("queryError has empty SourceName: %v", e)
	}

	if e.ItemType == "" {
		return fmt.Errorf("queryError has empty ItemType: %v", e)
	}

	if e.ResponderName == "" {
		return fmt.Errorf("queryError has empty ResponderName: %v", e)
	}

	return nil
}

// Validate Ensures a Query is valid
func (q *Query) Validate() error {
	if q == nil {
		return errors.New("query is nil")
	}

	if q.Type == "" {
		return fmt.Errorf("query has empty Type: %v", q)
	}

	if q.Scope == "" {
		return fmt.Errorf("query has empty Scope: %v", q)
	}

	if len(q.UUID) == 0 {
		return fmt.Errorf("query has empty UUID: %v", q)
	}

	if q.Method == QueryMethod_GET {
		if q.Query == "" {
			return fmt.Errorf("query cannot have empty Query when method is Get: %v", q)
		}
	}

	return nil
}
