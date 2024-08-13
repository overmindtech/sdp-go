package sdp

import "math"

// CalculatePaginationOffsetLimit Calculates the offset and limit for pagination
// in SQL queries, along with the current page and total pages that should be
// included in the response
func CalculatePaginationOffsetLimit(pagination *PaginationRequest, totalItems int32) (offset, limit, page, totalPages int32) {
	// pagesize is at least 10, at most 100
	limit = min(100, max(10, pagination.GetPageSize()))
	// calculate the total number of pages
	totalPages = int32(math.Ceil(float64(totalItems) / float64(limit)))

	// page has to be at least 1, and at most totalPages
	page = min(totalPages, pagination.GetPage())
	page = max(1, page)

	// calculate the offset
	if totalPages == 0 {
		offset = 0
	} else {
		offset = (page * limit) - limit
	}
	return offset, limit, page, totalPages
}
