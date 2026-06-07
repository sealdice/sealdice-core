package tooltest

import (
	"sealdice-core/model/common/response"
)

type PostMessageReqBody struct {
	Text string `json:"text"`
	Mode string `json:"mode"`
}

type PostMessageReq struct {
	Body PostMessageReqBody `json:"body"`
}

type MessageItem struct {
	UID         string `json:"uid"`
	Message     string `json:"message"`
	MessageType string `json:"messageType"`
}

type PendingMessagesResp struct {
	Items []MessageItem `json:"items"`
}

type CommandsResp struct {
	Items []string `json:"items"`
}

type PendingMessagesItemResponse = response.ItemResponse[PendingMessagesResp]
type CommandsItemResponse = response.ItemResponse[CommandsResp]
type SimpleItemResponse = response.ItemResponse[response.SimpleOK]
