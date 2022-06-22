package sdp

import "testing"

func TestEqual(t *testing.T) {
	x := &GatewayRequestStatus{
		ResponderStates: map[string]ResponderState{
			"foo": ResponderState_COMPLETE,
			"boo": ResponderState_WORKING,
			"bar": ResponderState_ERROR,
		},
		ResponderErrors: map[string]*ItemRequestError{
			"bar": {
				ErrorType:   ItemRequestError_TIMEOUT,
				Context:     "test",
				ErrorString: "timed out",
			},
		},
		Summary: &GatewayRequestStatus_Summary{
			Working:    1,
			Stalled:    0,
			Complete:   1,
			Error:      1,
			Cancelled:  0,
			Responders: 3,
		},
	}

	t.Run("with nil summary", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			ResponderErrors: map[string]*ItemRequestError{
				"bar": {
					ErrorType:   ItemRequestError_TIMEOUT,
					Context:     "test",
					ErrorString: "timed out",
				},
			},
		}

		if x.Equal(y) {
			t.Error("expected items to be nonequal")
		}
	})

	t.Run("with nil ResponderErrors", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
		}

		if x.Equal(y) {
			t.Error("expected items to be nonequal")
		}
	})

	t.Run("with nil ResponderStates", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderErrors: map[string]*ItemRequestError{
				"bar": {
					ErrorType:   ItemRequestError_TIMEOUT,
					Context:     "test",
					ErrorString: "timed out",
				},
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
		}

		if x.Equal(y) {
			t.Error("expected items to be nonequal")
		}
	})

	t.Run("with mismatched summary", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			ResponderErrors: map[string]*ItemRequestError{
				"bar": {
					ErrorType:   ItemRequestError_TIMEOUT,
					Context:     "test",
					ErrorString: "timed out",
				},
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   3,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
		}

		if x.Equal(y) {
			t.Error("expected items to be nonequal")
		}
	})

	t.Run("with mismatched ResponderErrors", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			ResponderErrors: map[string]*ItemRequestError{},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
		}

		if x.Equal(y) {
			t.Error("expected items to be nonequal")
		}
	})

	t.Run("with mismatched ResponderStates", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"BOO": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			ResponderErrors: map[string]*ItemRequestError{
				"bar": {
					ErrorType:   ItemRequestError_TIMEOUT,
					Context:     "test",
					ErrorString: "timed out",
				},
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
		}

		if x.Equal(y) {
			t.Error("expected items to be nonequal")
		}
	})

	t.Run("with same everything", func(t *testing.T) {
		y := &GatewayRequestStatus{
			ResponderStates: map[string]ResponderState{
				"foo": ResponderState_COMPLETE,
				"boo": ResponderState_WORKING,
				"bar": ResponderState_ERROR,
			},
			ResponderErrors: map[string]*ItemRequestError{
				"bar": {
					ErrorType:   ItemRequestError_TIMEOUT,
					Context:     "test",
					ErrorString: "timed out",
				},
			},
			Summary: &GatewayRequestStatus_Summary{
				Working:    1,
				Stalled:    0,
				Complete:   1,
				Error:      1,
				Cancelled:  0,
				Responders: 3,
			},
		}

		if !x.Equal(y) {
			t.Error("expected items to be equal")
		}
	})
}
