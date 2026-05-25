package query

// CreateTabRequest is the request body for creating a new SQL Lab tab.
type CreateTabRequest struct {
	DbID       uint   `json:"db_id" binding:"required"`
	Schema     string `json:"schema"`
	Catalog    string `json:"catalog"`
	SQL        string `json:"sql"`
	QueryLimit int    `json:"query_limit"`
}

// UpdateTabRequest is the request body for updating a SQL Lab tab.
// All fields are optional — only non-nil fields are applied.
type UpdateTabRequest struct {
	Label         *string `json:"label"`
	SQL           *string `json:"sql"`
	Schema        *string `json:"schema"`
	Catalog       *string `json:"catalog"`
	QueryLimit    *int    `json:"query_limit"`
	DbID          *uint   `json:"db_id"`
	LatestQueryID *string `json:"latest_query_id"`
	HideLeftBar   *bool   `json:"hide_left_bar"`
	ExtraJSON     *string `json:"extra_json"`
}

// TabResponse is the API response for a SQL Lab tab.
type TabResponse struct {
	ID                uint   `json:"id"`
	Label             string `json:"label"`
	DbID              uint   `json:"db_id"`
	Schema            string `json:"schema"`
	Catalog           string `json:"catalog"`
	SQL               string `json:"sql"`
	Active            bool   `json:"active"`
	QueryLimit        int    `json:"query_limit"`
	LatestQueryID     *string `json:"latest_query_id,omitempty"`
	LatestQueryStatus string `json:"latest_query_status"`
	HideLeftBar       bool   `json:"hide_left_bar"`
	CreatedOn         string `json:"created_on"`
}

// CreateSavedQueryRequest is the request body for saving a query.
type CreateSavedQueryRequest struct {
	DbID        uint   `json:"db_id" binding:"required"`
	Label       string `json:"label" binding:"required"`
	Schema      string `json:"schema"`
	Catalog     string `json:"catalog"`
	SQL         string `json:"sql" binding:"required"`
	Description string `json:"description"`
	Published   *bool  `json:"published"`
}

// SavedQueryResponse is the API response for a saved query.
type SavedQueryResponse struct {
	ID          uint   `json:"id"`
	Label       string `json:"label"`
	DbID        uint   `json:"db_id"`
	Schema      string `json:"schema"`
	Catalog     string `json:"catalog"`
	SQL         string `json:"sql"`
	Description string `json:"description"`
	SQLTables   string `json:"sql_tables"`
	Published   bool   `json:"published"`
	CreatedOn   string `json:"created_on"`
	ChangedOn   string `json:"changed_on"`
}

// UpdateSavedQueryRequest is the request body for updating a saved query.
// All fields are optional — only non-nil fields are applied.
type UpdateSavedQueryRequest struct {
	Label       *string `json:"label"`
	SQL         *string `json:"sql"`
	Schema      *string `json:"schema"`
	Catalog     *string `json:"catalog"`
	Description *string `json:"description"`
	Published   *bool   `json:"published"`
	ExtraJSON   *string `json:"extra_json"`
}

// SavedQueryListParams holds query parameters for listing saved queries.
type SavedQueryListParams struct {
	Search    string `form:"q"`
	Published *bool  `form:"published"`
	Page      int    `form:"page"`
	Limit     int    `form:"limit"`
}
