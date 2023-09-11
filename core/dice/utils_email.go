package dice

import (
	"fmt"
	"net/smtp"
	"strings"
)

type MailCode int

const (
	// MailTypeOnebotClose 掉线
	MailTypeOnebotClose MailCode = iota
	// MailTypeCIAMLock 风控 // tx 云把 Customer Identity Access Management 叫做 账号风控平台……
	MailTypeCIAMLock
	// MailTypeNotice 通知
	MailTypeNotice
	// MailTypeSendNote send 指令
	MailTypeSendNote
)

func (d *Dice) SendMail(body string, m MailCode) {
	if d.MailFrom == "" || d.MailPassword == "" || d.MailSmtp == "" || !d.MailEnable {
		return
	}
	body += "\n自动邮件，无需回复。"
	sub := "Seal News!"
	switch m {
	case MailTypeOnebotClose:
		sub = "Onebot 连接中断"
	case MailTypeCIAMLock:
		sub = "Bot 机器人被风控"
	case MailTypeNotice:
		sub = "Event 事件通知"
	case MailTypeSendNote:
		sub = "Send 指令反馈"
	}
	var to []string
	for _, id := range d.NoticeIds {
		if strings.HasPrefix(id, "QQ:") {
			to = append(to, id[3:]+"@qq.com")
		}
		if strings.HasPrefix(id, "Mail:") {
			to = append(to, id[5:])
		}
	}
	// 固定格式
	// Please follow RFC5322, RFC2047, RFC822 standard protocol
	msg := fmt.Sprintf("Subject: %s\r\nFrom: %s\r\nTo: %s\r\n",
		sub,
		d.MailFrom, strings.Join(to, ";")) + "\n" + body
	// 强制 25 端口，因为 smtp.SendMail 不能处理 465 （SMTPS 协议）
	err := smtp.SendMail(d.MailSmtp+":25",
		smtp.PlainAuth("", d.MailFrom, d.MailPassword, d.MailSmtp),
		d.MailFrom, to, []byte(msg))
	if err != nil {
		d.Logger.Error(err)
	} else {
		d.Logger.Infof("Mail:[%s]%s -> %s", sub, body, strings.Join(to, ";"))
	}
}
