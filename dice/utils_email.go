package dice

import (
	"fmt"
	"strings"

	"gopkg.in/gomail.v2"
)

type MailCode int

const (
	// MailTypeConnectClose 掉线
	MailTypeConnectClose MailCode = iota
	// MailTypeCIAMLock 风控 // tx 云把 Customer Identity Access Management 叫做 账号风控平台……
	MailTypeCIAMLock
	// MailTypeNotice 通知
	MailTypeNotice
	// MailTypeSendNote send 指令
	MailTypeSendNote
	// MailTest 测试邮件
	MailTest
)

func (d *Dice) CanSendMail() bool {
	if d.MailFrom == "" || d.MailPassword == "" || d.MailSMTP == "" {
		return false
	}
	return true
}

func (d *Dice) SendMail(body string, m MailCode) error {
	if !d.CanSendMail() {
		return fmt.Errorf("邮件配置不完整")
	}
	sub := "Seal News: "
	switch m {
	case MailTypeConnectClose:
		sub += "Connect 连接中断"
	case MailTypeCIAMLock:
		sub += "Bot 机器人被风控"
	case MailTypeNotice:
		sub += "Event 事件通知"
	case MailTypeSendNote:
		sub += "Send 指令反馈"
	case MailTest:
		sub += "Test 测试邮件"
	}
	var to []string
	for _, id := range d.NoticeIDs {
		if strings.HasPrefix(id, "QQ:") {
			to = append(to, id[3:]+"@qq.com")
		}
		if strings.HasPrefix(id, "Mail:") {
			to = append(to, id[5:])
		}
	}

	d.SendMailRow(sub, to, body, nil)
	return nil
}

func (d *Dice) SendMailRow(subject string, to []string, content string, attachments []string) {
	m := gomail.NewMessage()
	// NOTE(Xiangze Li): 按理说应当统一用DiceFotmatTmpl, 但是那样还得有一个MsgContext, 好复杂
	diceName := "海豹核心"
	if v := d.TextMap["核心:骰子名字"]; v != nil {
		if s, ok := v.Pick().(string); ok {
			diceName = s
		}
	}
	m.SetHeader("Subject", fmt.Sprintf("[%s] %s", diceName, subject))
	m.SetHeader("From", d.MailFrom)
	m.SetHeader("To", to...)
	if content == "" {
		m.SetBody("text/plain", "***自动邮件，无需回复***")
	} else {
		m.SetBody("text/plain", content+"\n\n***自动邮件，无需回复***")
	}
	if len(attachments) > 0 {
		for _, attachment := range attachments {
			m.Attach(attachment)
		}
	}

	dialer := gomail.NewDialer(d.MailSMTP, 25, d.MailFrom, d.MailPassword)
	if err := dialer.DialAndSend(m); err != nil {
		d.Logger.Error(err)
	} else {
		d.Logger.Infof("Mail:[%s]%s -> %s", subject, content, strings.Join(to, ";"))
	}
}
