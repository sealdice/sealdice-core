package dice

import (
	"fmt"
	"path/filepath"
	"sealdice-core/utils"
)

type HTTPSimpleMessage struct {
	UID     string `json:"uid"`
	Message string `json:"message"`
}

type PlatformAdapterHTTP struct {
	RecentMessage []HTTPSimpleMessage
}

func (pa *PlatformAdapterHTTP) GetGroupInfoAsync(_ string) {}

func (pa *PlatformAdapterHTTP) Serve() int {
	return 0
}

func (pa *PlatformAdapterHTTP) DoRelogin() bool {
	return false
}

func (pa *PlatformAdapterHTTP) SetEnable(_ bool) {}

func (pa *PlatformAdapterHTTP) SendToPerson(_ *MsgContext, uid string, text string, _ string) {
	sp := utils.SplitLongText(text, 300)
	for _, sub := range sp {
		pa.RecentMessage = append(pa.RecentMessage, HTTPSimpleMessage{uid, sub})
	}
}

func (pa *PlatformAdapterHTTP) SendToGroup(_ *MsgContext, uid string, text string, _ string) {
	pa.SendToPerson(nil, uid, text, "")
}

func (pa *PlatformAdapterHTTP) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterHTTP) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterHTTP) QuitGroup(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterHTTP) SetGroupCardName(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterHTTP) MemberBan(_ string, _ string, _ int64) {}

func (pa *PlatformAdapterHTTP) MemberKick(_ string, _ string) {}
