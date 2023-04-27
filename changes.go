package sdp

import (
	"github.com/google/uuid"
)

func (a *AppMetadata) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *AppProperties) GetBookmarkUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetBookmarkUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *GetAppRequest) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *UpdateAppRequest) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *DeleteAppRequest) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *ListAppChangesRequest) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *ChangeMetadata) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *ChangeProperties) GetAffectedItemsBookmarkUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetAffectedItemsBookmarkUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *ChangeProperties) GetSystemBeforeSnapshotUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetSystemBeforeSnapshotUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *ChangeProperties) GetSystemAfterSnapshotUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetSystemAfterSnapshotUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *GetChangeRequest) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *UpdateChangeRequest) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (a *DeleteChangeRequest) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (ob *OnboardingProperties) GetAppUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(ob.GetAppUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (ob *OnboardingProperties) GetAwsSourceUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(ob.GetAwsSourceUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (ob *OnboardingProperties) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(ob.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}
