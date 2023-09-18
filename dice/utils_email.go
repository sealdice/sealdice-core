package dice

import (
	"strings"

	"gopkg.in/gomail.v2"
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

func (d *Dice) CanSendMail() bool {
	if d.MailFrom == "" || d.MailPassword == "" || d.MailSmtp == "" {
		return false
	}
	return true
}

func (d *Dice) SendMail(body string, m MailCode) {
	if !d.CanSendMail() {
		return
	}
	sub := "Seal News: "
	switch m {
	case MailTypeOnebotClose:
		sub += "Onebot 连接中断"
	case MailTypeCIAMLock:
		sub += "Bot 机器人被风控"
	case MailTypeNotice:
		sub += "Event 事件通知"
	case MailTypeSendNote:
		sub += "Send 指令反馈"
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

	d.SendMailRow(sub, to, body, nil)
}

func (d *Dice) SendMailRow(subject string, to []string, content string, attachments []string) {
	m := gomail.NewMessage()
	m.SetHeader("Subject", subject)
	m.SetHeader("From", d.MailFrom)
	m.SetHeader("To", to...)
	m.SetBody("text/plain", content+"\n\n自动邮件，无需回复。")
	if len(attachments) > 0 {
		for _, attachment := range attachments {
			m.Attach(attachment)
		}
	}

	dialer := gomail.NewDialer(d.MailSmtp, 25, d.MailFrom, d.MailPassword)
	if err := dialer.DialAndSend(m); err != nil {
		d.Logger.Error(err)
	} else {
		d.Logger.Infof("Mail:[%s]%s -> %s", subject, content, strings.Join(to, ";"))
	}
}
