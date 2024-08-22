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
		"metadata":   c.GetMetadata().ToMap(),
		"properties": c.GetProperties().ToMap(),
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
		"type":                 r.GetType(),
		"uniqueAttributeValue": r.GetUniqueAttributeValue(),
		"scope":                r.GetScope(),
	}
}

func (r *Risk) ToMap() map[string]any {
	relatedItems := make([]map[string]any, len(r.GetRelatedItems()))
	for i, ri := range r.GetRelatedItems() {
		relatedItems[i] = ri.ToMap()
	}

	return map[string]any{
		"uuid":         stringFromUuidBytes(r.GetUUID()),
		"title":        r.GetTitle(),
		"severity":     r.GetSeverity().String(),
		"description":  r.GetDescription(),
		"relatedItems": relatedItems,
	}
}

func (r *GetChangeRisksResponse) ToMap() map[string]any {
	rmd := r.GetChangeRiskMetadata()
	risks := make([]map[string]any, len(rmd.GetRisks()))
	for i, ri := range rmd.GetRisks() {
		risks[i] = ri.ToMap()
	}

	return map[string]any{
		"risks":                 risks,
		"numHighRisk":           rmd.GetNumHighRisk(),
		"numMediumRisk":         rmd.GetNumMediumRisk(),
		"numLowRisk":            rmd.GetNumLowRisk(),
		"riskCalculationStatus": rmd.GetRiskCalculationStatus().ToMap(),
	}
}

func (cm *ChangeMetadata) ToMap() map[string]any {
	return map[string]any{
		"UUID":                stringFromUuidBytes(cm.GetUUID()),
		"createdAt":           cm.GetCreatedAt().AsTime(),
		"updatedAt":           cm.GetUpdatedAt().AsTime(),
		"status":              cm.GetStatus().String(),
		"creatorName":         cm.GetCreatorName(),
		"numAffectedApps":     cm.GetNumAffectedApps(),
		"numAffectedItems":    cm.GetNumAffectedItems(),
		"numAffectedEdges":    cm.GetNumAffectedEdges(),
		"numUnchangedItems":   cm.GetNumUnchangedItems(),
		"numCreatedItems":     cm.GetNumCreatedItems(),
		"numUpdatedItems":     cm.GetNumUpdatedItems(),
		"numDeletedItems":     cm.GetNumDeletedItems(),
		"UnknownHealthChange": cm.GetUnknownHealthChange(),
		"OkHealthChange":      cm.GetOkHealthChange(),
		"WarningHealthChange": cm.GetWarningHealthChange(),
		"ErrorHealthChange":   cm.GetErrorHealthChange(),
		"PendingHealthChange": cm.GetPendingHealthChange(),
	}
}

func (i *Item) ToMap() map[string]any {
	return map[string]any{
		"type":                 i.GetType(),
		"uniqueAttributeValue": i.UniqueAttributeValue(),
		"scope":                i.GetScope(),
		"attributes":           i.GetAttributes().GetAttrStruct().GetFields(),
	}
}

func (id *ItemDiff) ToMap() map[string]any {
	result := map[string]any{
		"status": id.GetStatus().String(),
	}
	if id.GetItem() != nil {
		result["item"] = id.GetItem().ToMap()
	}
	if id.GetBefore() != nil {
		result["before"] = id.GetBefore().ToMap()
	}
	if id.GetAfter() != nil {
		result["after"] = id.GetAfter().ToMap()
	}
	return result
}

func (id *ItemDiff) GloballyUniqueName() string {
	if id.GetItem() != nil {
		return id.GetItem().GloballyUniqueName()
	} else if id.GetBefore() != nil {
		return id.GetBefore().GloballyUniqueName()
	} else if id.GetAfter() != nil {
		return id.GetAfter().GloballyUniqueName()
	} else {
		return "empty item diff"
	}
}

func (cp *ChangeProperties) ToMap() map[string]any {
	affectedApps := make([]string, len(cp.GetAffectedAppsUUID()))
	for i, u := range cp.GetAffectedAppsUUID() {
		affectedApps[i] = stringFromUuidBytes(u)
	}
	plannedChanges := make([]map[string]any, len(cp.GetPlannedChanges()))
	for i, id := range cp.GetPlannedChanges() {
		plannedChanges[i] = id.ToMap()
	}

	return map[string]any{
		"title":                     cp.GetTitle(),
		"description":               cp.GetDescription(),
		"ticketLink":                cp.GetTicketLink(),
		"owner":                     cp.GetOwner(),
		"ccEmails":                  cp.GetCcEmails(),
		"changingItemsBookmarkUUID": stringFromUuidBytes(cp.GetChangingItemsBookmarkUUID()),
		"blastRadiusSnapshotUUID":   stringFromUuidBytes(cp.GetBlastRadiusSnapshotUUID()),
		"systemBeforeSnapshotUUID":  stringFromUuidBytes(cp.GetSystemBeforeSnapshotUUID()),
		"systemAfterSnapshotUUID":   stringFromUuidBytes(cp.GetSystemAfterSnapshotUUID()),
		"affectedAppsUUID":          cp.GetAffectedAppsUUID(),
		"plannedChanges":            cp.GetPlannedChanges(),
		"rawPlan":                   cp.GetRawPlan(),
		"codeChanges":               cp.GetCodeChanges(),
	}
}

func (rcs *RiskCalculationStatus) ToMap() map[string]any {
	if rcs == nil {
		return map[string]any{}
	}

	milestones := make([]map[string]any, len(rcs.GetProgressMilestones()))

	for i, milestone := range rcs.GetProgressMilestones() {
		milestones[i] = milestone.ToMap()
	}

	return map[string]any{
		"status":             rcs.GetStatus().String(),
		"progressMilestones": milestones,
	}
}

func (m *RiskCalculationStatus_ProgressMilestone) ToMap() map[string]any {
	return map[string]any{
		"description": m.GetDescription(),
		"status":      m.GetStatus().String(),
	}
}

func (s CalculateBlastRadiusResponse_State) ToMessage() string {
	switch s {
	case CalculateBlastRadiusResponse_STATE_UNSPECIFIED:
		return "unknown"
	case CalculateBlastRadiusResponse_STATE_DISCOVERING:
		return "The blast radius is being calculated"
	case CalculateBlastRadiusResponse_STATE_SAVING:
		return "The blast radius has been calculated and is being saved"
	case CalculateBlastRadiusResponse_STATE_FINDING_APPS:
		return "Determining which apps are within the blast radius"
	case CalculateBlastRadiusResponse_STATE_DONE:
		return "The blast radius calculation is complete"
	default:
		return "unknown"
	}
}

func (s StartChangeResponse_State) ToMessage() string {
	switch s {
	case StartChangeResponse_STATE_UNSPECIFIED:
		return "unknown"
	case StartChangeResponse_STATE_TAKING_SNAPSHOT:
		return "Snapshot is being taken"
	case StartChangeResponse_STATE_SAVING_SNAPSHOT:
		return "Snapshot is being saved"
	case StartChangeResponse_STATE_DONE:
		return "Everything is complete"
	default:
		return "unknown"
	}
}

func (s EndChangeResponse_State) ToMessage() string {
	switch s {
	case EndChangeResponse_STATE_UNSPECIFIED:
		return "unknown"
	case EndChangeResponse_STATE_TAKING_SNAPSHOT:
		return "Snapshot is being taken"
	case EndChangeResponse_STATE_SAVING_SNAPSHOT:
		return "Snapshot is being saved"
	case EndChangeResponse_STATE_DONE:
		return "Everything is complete"
	default:
		return "unknown"
	}
}
