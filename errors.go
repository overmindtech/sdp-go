package sdp

import "fmt"

const ErrorTemplate string = `%v

ErrorType: %v
Context: %v
SourceName: %v
ItemType: %v
ResponderName: %v`

// Ensure that the ItemRequestError is seen as a valid error in golang
func (e *ItemRequestError) Error() string {
	return fmt.Sprintf(
		ErrorTemplate,
		e.ErrorString,
		e.ErrorType.String(),
		e.Context,
		e.SourceName,
		e.ItemType,
		e.ResponderName,
	)
}

// NewItemRequestError converts a regular error to an ItemRequestError of type
// OTHER. If the input error is already an ItemRequestError then it is preserved
func NewItemRequestError(err error) *ItemRequestError {
	if sdpErr, ok := err.(*ItemRequestError); ok {
		return sdpErr
	}

	return &ItemRequestError{
		ErrorType:   ItemRequestError_OTHER,
		ErrorString: err.Error(),
	}
}
