package schema

// fork from https://github.com/nsxdevx/nsxbot

import (
	"github.com/bytedance/sonic"
)

type Message struct {
	Type string                 `json:"type"`
	Data sonic.NoCopyRawMessage `json:"data"`
}

type Text struct {
	Text string `json:"text"`
}

type Face struct {
	Id string `json:"id"`
}

type At struct {
	QQ string `json:"qq"`
}

type Reply struct {
	Id int `json:"id"`
}

type CommonFile struct {
	File string `json:"file"`
	Url  string `json:"url,omitzero"`
	Path string `json:"path,omitzero"`

	FileID     string `json:"file_id,omitzero"`
	FileSize   string `json:"file_size,omitzero"`
	FileUnique string `json:"file_unique,omitzero"`
}

type Image struct {
	CommonFile
	Type    string `json:"type,omitzero"`
	Summary string `json:"summary,omitzero"`
	SubType int    `json:"sub_type,omitzero"`
}

type Record struct {
	CommonFile
	// The magic field is generally not implemented (even in go-cqhttp) because there is insufficient demand
	Magic bool `json:"magic"`
}
