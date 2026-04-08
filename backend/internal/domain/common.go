package domain

// ListParams contains common pagination parameters
type ListParams struct {
	Limit           *int
	Page            *string
	IncludeArchived bool
	WorkspaceID     string
}

// ListResponse is the standard envelope for paginated list responses
type ListResponse[T any] struct {
	Data     []T     `json:"data"`
	NextPage *string `json:"next_page"`
}

func DefaultLimit(limit *int, defaultVal int) int {
	if limit == nil || *limit <= 0 {
		return defaultVal
	}
	if *limit > 100 {
		return 100
	}
	return *limit
}
