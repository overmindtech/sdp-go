package sdp

func (s *Snapshot) ToMap() map[string]any {
	return map[string]any{
		"metadata":   s.Metadata.ToMap(),
		"properties": s.Properties.ToMap(),
	}
}

func (sm *SnapshotMetadata) ToMap() map[string]any {
	return map[string]any{
		"UUID":    stringFromUuidBytes(sm.UUID),
		"created": sm.Created.AsTime(),
	}
}

func (sp *SnapshotProperties) ToMap() map[string]any {
	return map[string]any{
		"name":          sp.Name,
		"description":   sp.Description,
		"queries":       sp.Queries,
		"excludedItems": sp.ExcludedItems,
		"Items":         sp.Items,
	}
}
