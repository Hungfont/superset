package chart

type CreateChartRequest struct {
	SliceName            string `json:"slice_name" binding:"required,max=255"`
	VizType              string `json:"viz_type" binding:"required"`
	DatasourceID         string `json:"datasource_id" binding:"required"`
	DatasourceType       string `json:"datasource_type" binding:"required"`
	Params               string `json:"params"`
	QueryContext         string `json:"query_context"`
	Description          string `json:"description"`
	CacheTimeout         int    `json:"cache_timeout"`
	CertifiedBy          string `json:"certified_by"`
	CertificationDetails string `json:"certification_details"`
}
