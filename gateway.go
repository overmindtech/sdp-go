package sdp

//Equal Returns whether to statuses are functionally equal
func (x *GatewayRequestStatus) Equal(y *GatewayRequestStatus) bool {
	if x == nil {
		if y == nil {
			return true
		} else {
			return false
		}
	}

	// Check the basics first
	if len(x.ResponderErrors) != len(y.ResponderErrors) {
		return false
	}

	if len(x.ResponderStates) != len(y.ResponderStates) {
		return false
	}

	if (x.Summary == nil || y.Summary == nil) && x.Summary != y.Summary {
		// If one of them is nil, and they aren't both nil
		return false
	}

	for xResponder, xErr := range x.ResponderErrors {
		xErrString := xErr.Error()

		yErr, exists := y.ResponderErrors[xResponder]

		if !exists {
			return false
		}

		if yErr.Error() != xErrString {
			return false
		}
	}

	for xResponder, xState := range x.ResponderStates {
		yState, exists := y.ResponderStates[xResponder]

		if !exists {
			return false
		}

		if yState != xState {
			return false
		}
	}

	if x.Summary != nil && y.Summary != nil {
		if x.Summary.Working != y.Summary.Working {
			return false
		}
		if x.Summary.Stalled != y.Summary.Stalled {
			return false
		}
		if x.Summary.Complete != y.Summary.Complete {
			return false
		}
		if x.Summary.Error != y.Summary.Error {
			return false
		}
		if x.Summary.Cancelled != y.Summary.Cancelled {
			return false
		}
		if x.Summary.Responders != y.Summary.Responders {
			return false
		}

	}

	return true
}
