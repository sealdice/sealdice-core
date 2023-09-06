package dice

import (
	"fmt"
	"net/smtp"
	"strings"
)

type MailCode int

const (
	// 掉线
	OnebotClose MailCode = iota
	// 风控 // tx 云把 Customer Identity Access Management 叫做 账号风控平台……
	CIAMLock
	// 通知
	Notice
	// send 指令
	SendNote
)

func (d *Dice) SendMail(body string, m MailCode) {
	if d.MailFrom == "" || d.MailPassword == "" || d.MailSmtp == "" || !d.MailEnable {
		return
	}
	body += "\n自动邮件，无需回复。"
	sub := "Seal News!"
	switch m {
	case OnebotClose:
		sub = "Onebot 连接中断"
	case CIAMLock:
		sub = "Bot 机器人被风控"
	case Notice:
		sub = "Event 事件通知"
	case SendNote:
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
