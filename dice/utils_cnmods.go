package dice

import (
	"log"
	"strconv"

	"github.com/monaco-io/request"
)

type CnmodsDetailInfo struct {
	Code      int   `json:"code"`
	Timestamp int64 `json:"timestamp"`
	Data      struct {
		Module struct {
			KeyID           int    `json:"keyId"`
			CreateTime      string `json:"createTime"`
			Sort            int    `json:"sort"`
			Title           string `json:"title"`
			ReleaseDate     string `json:"releaseDate"`
			ModuleType      string `json:"moduleType"`
			ModuleVersion   string `json:"moduleVersion"`
			ModuleAge       string `json:"moduleAge"`
			FreeLevel       string `json:"freeLevel"`
			OccurrencePlace string `json:"occurrencePlace"`
			Structure       string `json:"structure"`
			MinDuration     int    `json:"minDuration"`
			MaxDuration     int    `json:"maxDuration"`
			MinAmount       int    `json:"minAmount"`
			MaxAmount       int    `json:"maxAmount"`
			Article         string `json:"article"`
			URL             string `json:"url"`
			Email           string `json:"email"`
			Command         bool   `json:"command"`
			Opinion         string `json:"opinion"`
			Original        bool   `json:"original"`
			Qq              string `json:"qq"`
			CustomerID      int    `json:"customerId"`
			OpenComment     bool   `json:"openComment"`
			Open            bool   `json:"open"`
		} `json:"module"`
		ModuleToLabelList []interface{} `json:"moduleToLabelList"`
		RelatedModuleList []interface{} `json:"relatedModuleList"`
		RecommendList     []struct {
			KeyID      int    `json:"keyId"`
			CreateTime string `json:"createTime"`
			Sort       int    `json:"sort"`
			Content    string `json:"content"`
			LoginUser  struct {
				KeyID      int    `json:"keyId"`
				CreateTime string `json:"createTime"`
				Sort       int    `json:"sort"`
				LoginID    string `json:"loginId"`
				Password   string `json:"password"`
				NickName   string `json:"nickName"`
			} `json:"loginUser"`
			Module struct {
				KeyID           int    `json:"keyId"`
				CreateTime      string `json:"createTime"`
				Sort            int    `json:"sort"`
				Title           string `json:"title"`
				ReleaseDate     string `json:"releaseDate"`
				ModuleType      string `json:"moduleType"`
				ModuleVersion   string `json:"moduleVersion"`
				ModuleAge       string `json:"moduleAge"`
				FreeLevel       string `json:"freeLevel"`
				OccurrencePlace string `json:"occurrencePlace"`
				Structure       string `json:"structure"`
				MinDuration     int    `json:"minDuration"`
				MaxDuration     int    `json:"maxDuration"`
				MinAmount       int    `json:"minAmount"`
				MaxAmount       int    `json:"maxAmount"`
				Article         string `json:"article"`
				URL             string `json:"url"`
				Email           string `json:"email"`
				Command         bool   `json:"command"`
				Opinion         string `json:"opinion"`
				Original        bool   `json:"original"`
				Qq              string `json:"qq"`
				CustomerID      int    `json:"customerId"`
				OpenComment     bool   `json:"openComment"`
				Open            bool   `json:"open"`
			} `json:"module"`
		} `json:"recommendList"`
		Collect bool   `json:"collect"`
		HeadPic string `json:"headPic"`
	} `json:"data"`
}

type CnmodsSearchResult struct {
	Code      int   `json:"code"`
	Timestamp int64 `json:"timestamp"`
	Data      struct {
		List []struct {
			Qq              string `json:"qq"`
			MinAmount       int    `json:"minAmount"`
			MinDuration     int    `json:"minDuration"`
			Original        bool   `json:"original"`
			ModuleType      string `json:"moduleType"`
			ReleaseDate     string `json:"releaseDate"`
			KeyID           int    `json:"keyId"`
			Sort            int    `json:"sort"`
			Title           string `json:"title"`
			Structure       string `json:"structure"`
			Article         string `json:"article"`
			URL             string `json:"url"`
			ModuleAge       string `json:"moduleAge"`
			FreeLevel       string `json:"freeLevel"`
			Opinion         string `json:"opinion"`
			UpdateLastWeek  bool   `json:"updateLastWeek"`
			ModuleVersion   string `json:"moduleVersion"`
			OccurrencePlace string `json:"occurrencePlace"`
			MaxAmount       int    `json:"maxAmount"`
			MaxDuration     int    `json:"maxDuration"`
			Email           string `json:"email"`
		} `json:"list"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
	} `json:"data"`
}

func CnmodsSearch(title string, page int, pageSize int, isRec bool, article string) *CnmodsSearchResult {
	var result CnmodsSearchResult
	query := map[string]string{
		"title":    title,
		"page":     strconv.Itoa(page),
		"pageSize": strconv.Itoa(pageSize), // 12
	}
	if isRec {
		query["command"] = "true"
	}
	if article != "" {
		query["title"] = ""
		query["article"] = article
	}

	// moduleType: DND COC
	// moduleVersion: coc7th coc6th
	c := request.Client{
		URL:    "https://www.cnmods.net/prod-api/index/moduleListPage.do",
		Method: "GET",
		Header: map[string]string{
			"Referer":    "https://www.cnmods.net/web/",
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36 Edg/100.0.1185.50",
		},
		Query: query,
	}
	resp := c.Send().Scan(&result)
	if !resp.OK() {
		// handle error
		log.Println(resp.Error())
	} else {
		return &result
	}
	return nil
}

func CnmodsDetail(keyID string) *CnmodsDetailInfo {
	var result CnmodsDetailInfo
	query := map[string]string{
		"keyId": keyID,
	}

	// moduleType: DND COC
	// moduleVersion: coc7th coc6th
	c := request.Client{
		URL:    "https://www.cnmods.net/prod-api/index/moduleDetail.do",
		Method: "GET",
		Header: map[string]string{
			"Referer":    "https://www.cnmods.net/web/",
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36 Edg/100.0.1185.50",
		},
		Query: query,
	}
	resp := c.Send().Scan(&result)
	if !resp.OK() {
		// handle error
		log.Println(resp.Error())
	} else {
		return &result
	}
	return nil
}
