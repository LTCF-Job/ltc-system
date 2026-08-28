package handler

// CreateSiteRequest 代表新增據點請求，欄位需求與既有 repository.SiteEntity binding 行為一致（無強制必填）。
type CreateSiteRequest struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	Region   string  `json:"region"`
	OpenDays []int16 `json:"openDays"`
	Status   string  `json:"status"`
}

// UpdateSiteRequest 代表更新據點請求。
type UpdateSiteRequest struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	Region   string  `json:"region"`
	OpenDays []int16 `json:"openDays"`
	Status   string  `json:"status"`
}
