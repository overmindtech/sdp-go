package sdp

func (b *Bookmark) ToMap() map[string]any {
	return map[string]any{
		"metadata":   b.Metadata.ToMap(),
		"properties": b.Properties.ToMap(),
	}
}

func (bm *BookmarkMetadata) ToMap() map[string]any {
	return map[string]any{
		"UUID":    stringFromUuidBytes(bm.UUID),
		"created": bm.Created.AsTime(),
	}
}

func (bp *BookmarkProperties) ToMap() map[string]any {
	return map[string]any{
		"name":          bp.Name,
		"description":   bp.Description,
		"queries":       bp.Queries,
		"excludedItems": bp.ExcludedItems,
	}
}
