package sdp

import "errors"

// Validate Ensures that en item is valid (e.g. contains the required fields)
func (i *Item) Validate() error {
	if i == nil {
		return errors.New("Item is nil")
	}

	if i.Type == "" {
		return errors.New("item has empty Type")
	}

	if i.UniqueAttribute == "" {
		return errors.New("item has empty UniqueAttribute")
	}

	if i.Attributes == nil {
		return errors.New("item has nil Attributes")
	}

	if i.Scope == "" {
		return errors.New("item has empty Scope")
	}

	if i.UniqueAttributeValue() == "" {
		return errors.New("item has empty UniqueAttributeValue")
	}

	return nil
}

// Validate Ensures a reference is valid
func (r *Reference) Validate() error {
	if r == nil {
		return errors.New("Reference is nil")
	}

	if r.Type == "" {
		return errors.New("reference has empty Type")
	}
	if r.UniqueAttributeValue == "" {
		return errors.New("reference has empty UniqueAttributeValue")
	}
	if r.Scope == "" {
		return errors.New("reference has empty Scope")
	}

	return nil
}

// Validate Ensures an edge is valid
func (e *Edge) Validate() error {
	if e == nil {
		return errors.New("Edge is nil")
	}

	var err error

	err = e.From.Validate()

	if err != nil {
		return err
	}

	err = e.To.Validate()

	return err
}

// Validate Ensures a ReverseLinksRequest is valid
func (r *ReverseLinksRequest) Validate() error {
	if r == nil {
		return errors.New("ReverseLinksRequest is nil")
	}

	if r.Item == nil {
		return errors.New("ReverseLinksRequest cannot have nil Item")
	} else {
		err := r.Item.Validate()

		if err != nil {
			return err
		}
	}

	return nil
}

// Validate Ensures a Response is valid
func (r *Response) Validate() error {
	if r == nil {
		return errors.New("Response is nil")
	}

	if r.Responder == "" {
		return errors.New("Response has empty Responder")
	}

	if len(r.ItemRequestUUID) == 0 {
		return errors.New("Response has empty ItemRequestUUID")
	}

	return nil
}

// Validate Ensures an ItemRequestError is valid
func (e *ItemRequestError) Validate() error {
	if e == nil {
		return errors.New("ItemRequestError is nil")
	}

	if len(e.ItemRequestUUID) == 0 {
		return errors.New("ItemRequestError has empty ItemRequestUUID")
	}

	if e.ErrorString == "" {
		return errors.New("ItemRequestError has empty ErrorString")
	}

	if e.Scope == "" {
		return errors.New("ItemRequestError has empty Scope")
	}

	if e.SourceName == "" {
		return errors.New("ItemRequestError has empty SourceName")
	}

	if e.ItemType == "" {
		return errors.New("ItemRequestError has empty ItemType")
	}

	if e.ResponderName == "" {
		return errors.New("ItemRequestError has empty ResponderName")
	}

	return nil
}

// Validate Ensures an ItemRequest is valid
func (r *ItemRequest) Validate() error {
	if r == nil {
		return errors.New("ItemRequest is nil")
	}

	if r.Type == "" {
		return errors.New("ItemRequest has empty Type")
	}

	if r.Scope == "" {
		return errors.New("ItemRequest has empty Scope")
	}

	if len(r.UUID) == 0 {
		return errors.New("Response has empty UUID")
	}

	if r.Method == RequestMethod_GET {
		if r.Query == "" {
			return errors.New("ItemRequest cannot have empty Query when method is Get")
		}
	}

	return nil
}
