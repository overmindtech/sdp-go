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

func (x *GetChangeTimelineRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *GetDiffRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *ListChangingItemsSummaryRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *GetAffectedAppsRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *UpdateChangingItemsRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *UpdatePlannedChangesRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *CalculateBlastRadiusRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *StartChangeRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *EndChangeRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *SimulateChangeRequest) GetChangeUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetChangeUUID())
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

func (r *Reference) ToMap() map[string]any {
	return map[string]any{
		"type":                 r.Type,
		"uniqueAttributeValue": r.UniqueAttributeValue,
		"scope":                r.Scope,
	}
}

func (r *Risk) ToMap() map[string]any {
	relatedItems := make([]map[string]any, len(r.RelatedItems))
	for i, ri := range r.RelatedItems {
		relatedItems[i] = ri.ToMap()
	}

	return map[string]any{
		"uuid":         stringFromUuidBytes(r.UUID),
		"title":        r.Title,
		"severity":     r.Severity.String(),
		"description":  r.Description,
		"relatedItems": relatedItems,
	}
}

func (r *GetChangeRisksResponse) ToMap() map[string]any {
	rmd := r.GetChangeRiskMetadata()
	risks := make([]map[string]any, len(rmd.Risks))
	for i, ri := range rmd.Risks {
		risks[i] = ri.ToMap()
	}

	return map[string]any{
		"risks":                 risks,
		"numHighRisk":           rmd.NumHighRisk,
		"numMediumRisk":         rmd.NumMediumRisk,
		"numLowRisk":            rmd.NumLowRisk,
		"riskCalculationStatus": rmd.RiskCalculationStatus.ToMap(),
	}
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
		"numAffectedEdges":    cm.NumAffectedEdges,
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

func (i *Item) ToMap() map[string]any {
	return map[string]any{
		"type":                 i.Type,
		"uniqueAttributeValue": i.UniqueAttributeValue(),
		"scope":                i.Scope,
		"attributes":           i.Attributes.AttrStruct.Fields,
	}
}

func (id *ItemDiff) ToMap() map[string]any {
	result := map[string]any{
		"status": id.Status.String(),
	}
	if id.Item != nil {
		result["item"] = id.Item.ToMap()
	}
	if id.Before != nil {
		result["before"] = id.Before.ToMap()
	}
	if id.After != nil {
		result["after"] = id.After.ToMap()
	}
	return result
}

func (id *ItemDiff) GloballyUniqueName() string {
	if id.Item != nil {
		return id.Item.GloballyUniqueName()
	} else if id.Before != nil {
		return id.Before.GloballyUniqueName()
	} else if id.After != nil {
		return id.After.GloballyUniqueName()
	} else {
		return "empty item diff"
	}
}

func (cp *ChangeProperties) ToMap() map[string]any {
	affectedApps := make([]string, len(cp.AffectedAppsUUID))
	for i, u := range cp.AffectedAppsUUID {
		affectedApps[i] = stringFromUuidBytes(u)
	}
	plannedChanges := make([]map[string]any, len(cp.PlannedChanges))
	for i, id := range cp.PlannedChanges {
		plannedChanges[i] = id.ToMap()
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
		"plannedChanges":            plannedChanges,
		"rawPlan":                   cp.RawPlan,
		"codeChanges":               cp.CodeChanges,
	}
}

func (rcs *RiskCalculationStatus) ToMap() map[string]any {
	if rcs == nil {
		return map[string]any{}
	}

	milestones := make([]map[string]any, len(rcs.ProgressMilestones))

	for i, milestone := range rcs.ProgressMilestones {
		milestones[i] = milestone.ToMap()
	}

	return map[string]any{
		"status":             rcs.Status.String(),
		"progressMilestones": milestones,
	}
}

func (m *RiskCalculationStatus_ProgressMilestone) ToMap() map[string]any {
	return map[string]any{
		"description": m.Description,
		"status":      m.Status.String(),
	}
}
