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

// ── Schema Browser (SQL-006) ──

// ExpandTableRequest is the body for POST /api/v1/sqllab/tabs/:id/schema
type ExpandTableRequest struct {
	TableName string `json:"table_name" binding:"required"`
}

// SchemaColumnItem is a column in the schema browser response.
type SchemaColumnItem struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   bool   `json:"is_nullable"`
	DefaultValue string `json:"default_value,omitempty"`
	IsDttm       bool   `json:"is_dttm"`
}

// SchemaTableItem is one table row in the GET schema response.
type SchemaTableItem struct {
	TableName string             `json:"table_name"`
	TableType string             `json:"table_type"`
	Expanded  bool               `json:"expanded"`
	Columns   []SchemaColumnItem `json:"columns,omitempty"`
}

// ExpandTableResponse is returned by POST expand.
type ExpandTableResponse struct {
	TableName string             `json:"table_name"`
	Columns   []SchemaColumnItem `json:"columns"`
}

// GetSchemaResponse is returned by GET /api/v1/sqllab/tabs/:id/schema
type GetSchemaResponse struct {
	Schemas []string          `json:"schemas"`
	Tables  []SchemaTableItem `json:"tables"`
}

// ── Autocomplete (SQL-007) ──

// AutocompleteRequest is the body for POST /api/v1/sqllab/autocomplete.
type AutocompleteRequest struct {
	Word   string `json:"word" binding:"required"`
	Prefix string `json:"prefix"`
	DbID   uint   `json:"db_id"`
	Schema string `json:"schema"`
}

// AutocompleteSuggestion is a single suggestion item.
type AutocompleteSuggestion struct {
	Text   string `json:"text"`
	Type   string `json:"type"` // "keyword"|"schema"|"table"|"column"|"function"
	Score  int    `json:"score"`
	Detail string `json:"detail"`
}

// AutocompleteResponse is the API response for autocomplete.
type AutocompleteResponse struct {
	Suggestions []AutocompleteSuggestion `json:"suggestions"`
	CacheMiss   bool                     `json:"cache_miss"`
}
