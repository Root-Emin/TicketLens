package dto

// Pagination bounds from the API contract: ?page=1&page_size=25, page_size max 100.
const (
	DefaultPage     = 1
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// PageParams holds validated pagination input.
type PageParams struct {
	Page     int
	PageSize int
}

// Offset returns the SQL offset value.
func (p PageParams) Offset() int { return (p.Page - 1) * p.PageSize }

// Limit returns the SQL limit value.
func (p PageParams) Limit() int { return p.PageSize }

// ListMeta is the pagination envelope the contract specifies. It differs from
// the template's shared/pagination shape (page/per_page/total_count, flat) —
// the contract is the source of truth for these endpoints because the frontend
// client is generated from it.
type ListMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// ListResponse wraps a paginated collection.
type ListResponse[T any] struct {
	Data []T      `json:"data"`
	Meta ListMeta `json:"meta"`
}

// NewListResponse builds a paginated response, rendering an empty collection as
// [] rather than null.
func NewListResponse[T any](data []T, params PageParams, total int) ListResponse[T] {
	if data == nil {
		data = []T{}
	}
	return ListResponse[T]{
		Data: data,
		Meta: ListMeta{Page: params.Page, PageSize: params.PageSize, Total: total},
	}
}

// CollectionResponse wraps an unpaginated collection (e.g. GET /departments).
type CollectionResponse[T any] struct {
	Data []T `json:"data"`
}

// NewCollectionResponse builds an unpaginated response.
func NewCollectionResponse[T any](data []T) CollectionResponse[T] {
	if data == nil {
		data = []T{}
	}
	return CollectionResponse[T]{Data: data}
}
