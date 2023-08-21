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

func (a *ChangeProperties) GetChangingItemsBookmarkUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(a.GetChangingItemsBookmarkUUID())
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

func (ci *UpdateChangingItemsRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(ci.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (c *Change) ToMap() map[string]any {
	return map[string]any{
		"metadata":   c.Metadata.ToMap(),
		"properties": c.Properties.ToMap(),
	}
}

func stringFromUuidBytes(b []byte) string {
	u, err := uuid.FromBytes(b)
	if err != nil {
		return ""
	}
	return u.String()
}

func (cm *ChangeMetadata) ToMap() map[string]any {
	return map[string]any{
		"UUID":                stringFromUuidBytes(cm.UUID),
		"createdAt":           cm.CreatedAt.AsTime(),
		"updatedAt":           cm.UpdatedAt.AsTime(),
		"status":              cm.Status.String(),
		"creatorName":         cm.CreatorName,
		"numAffectedApps":     cm.NumAffectedApps,
		"numAffectedItems":    cm.NumAffectedItems,
		"numUnchangedItems":   cm.NumUnchangedItems,
		"numCreatedItems":     cm.NumCreatedItems,
		"numUpdatedItems":     cm.NumUpdatedItems,
		"numDeletedItems":     cm.NumDeletedItems,
		"UnknownHealthChange": cm.UnknownHealthChange,
		"OkHealthChange":      cm.OkHealthChange,
		"WarningHealthChange": cm.WarningHealthChange,
		"ErrorHealthChange":   cm.ErrorHealthChange,
		"PendingHealthChange": cm.PendingHealthChange,
	}
}

func (cp *ChangeProperties) ToMap() map[string]any {
	affectedApps := []string{}
	for _, u := range cp.AffectedAppsUUID {
		affectedApps = append(affectedApps, stringFromUuidBytes(u))
	}

	return map[string]any{
		"title":                     cp.Title,
		"description":               cp.Description,
		"ticketLink":                cp.TicketLink,
		"owner":                     cp.Owner,
		"ccEmails":                  cp.CcEmails,
		"changingItemsBookmarkUUID": stringFromUuidBytes(cp.ChangingItemsBookmarkUUID),
		"blastRadiusSnapshotUUID":   stringFromUuidBytes(cp.BlastRadiusSnapshotUUID),
		"systemBeforeSnapshotUUID":  stringFromUuidBytes(cp.SystemBeforeSnapshotUUID),
		"systemAfterSnapshotUUID":   stringFromUuidBytes(cp.SystemAfterSnapshotUUID),
		"affectedAppsUUID":          affectedApps,
	}
}
