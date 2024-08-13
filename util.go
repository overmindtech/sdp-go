package sdp

import "math"

// CalculatePaginationOffsetLimit Calculates the offset and limit for pagination
// in SQL queries, along with the current page and total pages that should be
// included in the response
//
// This also sets sane defaults for the page size if pagination is not provided.
// These defaults are page 1 with a page size of 10
//
// NOTE: If there are no items, then this will return 0 for all values
func CalculatePaginationOffsetLimit(pagination *PaginationRequest, totalItems int32) (offset, limit, page, totalPages int32) {
	if totalItems == 0 {
		// If there are no items, there are no pages
		return 0, 0, 0, 0
	}

	var requestedPageSize int32
	var requestedPage int32

	if pagination == nil {
		// Set sane defaults
		requestedPageSize = 10
		requestedPage = 1
	} else {
		requestedPageSize = pagination.GetPageSize()
		requestedPage = pagination.GetPage()
	}

	// pagesize is at least 10, at most 100
	limit = min(100, max(10, requestedPageSize))
	// calculate the total number of pages
	totalPages = int32(math.Ceil(float64(totalItems) / float64(limit)))

	// page has to be at least 1, and at most totalPages
	page = min(totalPages, requestedPage)
	page = max(1, page)

	// calculate the offset
	if totalPages == 0 {
		offset = 0
	} else {
		offset = (page * limit) - limit
	}
	return offset, limit, page, totalPages
}
