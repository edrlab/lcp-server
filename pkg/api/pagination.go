package api

// PaginationKey is used to store pagination parameters in the context.
type PaginationKey string

const (
	PageKey    PaginationKey = "page"
	PerPageKey PaginationKey = "per_page"
)
