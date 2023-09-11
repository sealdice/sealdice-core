package dice

import (
	"fmt"
	"path/filepath"
)

type HttpSimpleMessage struct {
	Uid     string `json:"uid"`
	Message string `json:"message"`
}

type PlatformAdapterHttp struct {
	RecentMessage []HttpSimpleMessage
}

func (pa *PlatformAdapterHttp) GetGroupInfoAsync(groupId string) {}

func (pa *PlatformAdapterHttp) Serve() int {
	return 0
}

func (pa *PlatformAdapterHttp) DoRelogin() bool {
	return false
}

func (pa *PlatformAdapterHttp) SetEnable(enable bool) {}

func (pa *PlatformAdapterHttp) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	pa.RecentMessage = append(pa.RecentMessage, HttpSimpleMessage{uid, text})
}

func (pa *PlatformAdapterHttp) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	pa.RecentMessage = append(pa.RecentMessage, HttpSimpleMessage{uid, text})
}

func (pa *PlatformAdapterHttp) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterHttp) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterHttp) QuitGroup(ctx *MsgContext, id string) {}

func (pa *PlatformAdapterHttp) SetGroupCardName(groupId string, userId string, name string) {}

func (pa *PlatformAdapterHttp) MemberBan(groupId string, userId string, duration int64) {}

func (pa *PlatformAdapterHttp) MemberKick(groupId string, userId string) {}
