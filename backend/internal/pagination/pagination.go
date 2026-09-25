package pagination

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type Page[T any] struct {
	Data  []T   `json:"data"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}

func Normalize(page, limit int) (int, int, int) {
	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = DefaultLimit
	}

	if limit > MaxLimit {
		limit = MaxLimit
	}

	return page, limit, (page - 1) * limit
}
