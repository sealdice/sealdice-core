//nolint:testpackage
package dice

import (
	"errors"
	"net/textproto"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func newTestMailDice(address string) *Dice {
	d := &Dice{Logger: zap.NewNop().Sugar()}
	d.Config.MailSMTP = address
	d.Config.MailFrom = "sender@example.com"
	d.Config.MailPassword = "test-password"
	d.Config.NoticeIDs = []string{"Mail:recipient@example.com"}
	return d
}

func TestSendMailReturnsConfigurationError(t *testing.T) {
	d := newTestMailDice("smtp.qq.com:465:25")
	if err := d.SendMail("test", MailTest); err == nil || !strings.Contains(err.Error(), "SMTP 地址格式错误") {
		t.Fatalf("expected SMTP configuration error, got %v", err)
	}
}

func TestSendMailReturnsSMTPError(t *testing.T) {
	d := newTestMailDice(startRejectingSMTPServer(t))
	err := d.SendMail("test", MailTest)
	var smtpErr *textproto.Error
	if !errors.As(err, &smtpErr) || smtpErr.Code != 554 {
		t.Fatalf("expected SMTP 554 error from configured port, got %v", err)
	}
}
