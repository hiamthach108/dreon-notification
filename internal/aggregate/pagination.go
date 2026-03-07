package aggregate

type PaginationReq struct {
	Page     int     `form:"page" json:"page" validate:"gte=1"`
	PageSize int     `form:"pageSize" json:"pageSize" validate:"gte=1,lte=100"`
	Cursor   *string `form:"cursor" json:"cursor"`
}

type PaginationResp[T any] struct {
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	NextCursor string `json:"nextCursor,omitempty"`
	HasNext    bool   `json:"hasNext,omitempty"`
	Items      []T    `json:"items"`
}

type CursorPaginationReq struct {
	Cursor   *string `form:"cursor" json:"cursor"`
	PageSize int     `form:"pageSize" json:"pageSize" validate:"gte=1,lte=100"`
}

type CursorPaginationResp[T any] struct {
	Total      int64  `json:"total"`
	NextCursor string `json:"nextCursor,omitempty"`
	HasNext    bool   `json:"hasNext,omitempty"`
	Items      []T    `json:"items"`
}