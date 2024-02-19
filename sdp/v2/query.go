package sdp

// assert interface
var _ error = (*QueryError)(nil)

// implement the error interface for QueryError
func (e *QueryError) Error() string {
	switch e.ErrorType {
	case QueryError_ERROR_TYPE_PERMANENT_ERROR:
		return "permanent error: " + e.ErrorString
	case QueryError_ERROR_TYPE_TEMPORARY_ERROR:
		return "temporary error: " + e.ErrorString
	case QueryError_ERROR_TYPE_IMPLEMENTATION_ERROR:
		return "implementation error: " + e.ErrorString
	default:
		return e.ErrorString
	}
}
