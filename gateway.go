package sdp

import (
	"encoding/hex"

	"github.com/google/uuid"
)

// Equal Returns whether two statuses are functionally equal
func (x *GatewayRequestStatus) Equal(y *GatewayRequestStatus) bool {
	if x == nil {
		if y == nil {
			return true
		} else {
			return false
		}
	}

	// Check the basics first
	if len(x.ResponderStates) != len(y.ResponderStates) {
		return false
	}

	if (x.Summary == nil || y.Summary == nil) && x.Summary != y.Summary {
		// If one of them is nil, and they aren't both nil
		return false
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

	if x.PostProcessingComplete != y.PostProcessingComplete {
		return false
	}

	return true
}

// Whether the gateway request is complete
func (x *GatewayRequestStatus) Done() bool {
	return x.PostProcessingComplete && x.Summary.Working == 0
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *StoreBookmark) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err != nil {
			return ""
		}
		return u.String()
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *BookmarkStoreResult) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *LoadBookmark) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *BookmarkLoadResult) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *StoreSnapshot) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *SnapshotStoreResult) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *LoadSnapshot) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *SnapshotLoadResult) GetMsgIDLogString() string {
	bs := x.GetMsgID()
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 16 {
		u, err := uuid.FromBytes(bs)
		if err == nil {
			return u.String()
		}
	}
	return hex.EncodeToString(bs)
}

// GetMsgIDLogString returns the correlation ID as string for logging
func (x *QueryStatus) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}
