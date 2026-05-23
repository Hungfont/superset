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

// CloseAllTabsRequest is the request body for closing all tabs.
// ExceptID, when set, excludes that tab from being closed (used for "Close Others").
type CloseAllTabsRequest struct {
	ExceptID *uint `json:"except_id"`
}
