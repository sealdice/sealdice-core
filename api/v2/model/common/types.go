package common

type BatchDeleteResp struct {
	Fails []string `json:"fails"`
}

type SimpleOK struct {
	Success bool `json:"success"`
}

type IDListReq struct {
	IDs []string `json:"ids"`
}

type NameListReq struct {
	Names []string `json:"names"`
}
