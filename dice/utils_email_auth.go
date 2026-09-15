package dice

import (
	"errors"
	"net/smtp"
	"slices"
)

// requiredTLSAuth rejects unencrypted SMTP before net/smtp sends AUTH or credentials.
type requiredTLSAuth struct {
	host     string
	username string
	password string
	auth     smtp.Auth
	login    bool
}

func (a *requiredTLSAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	// gomail reuses Auth when reconnecting; recheck TLS on every connection.
	a.auth = nil
	a.login = false
	if !server.TLS {
		return "", nil, errors.New("SMTP 587 必须使用 STARTTLS，服务器未建立 TLS 连接，已拒绝发送邮件")
	}
	if server.Name != a.host {
		return "", nil, errors.New("SMTP 认证服务器名称不匹配")
	}

	// Preserve gomail's mechanism preference, but only after TLS is established.
	switch {
	case slices.Contains(server.Auth, "CRAM-MD5"):
		a.auth = smtp.CRAMMD5Auth(a.username, a.password)
	case slices.Contains(server.Auth, "PLAIN"):
		a.auth = smtp.PlainAuth("", a.username, a.password, a.host)
	case slices.Contains(server.Auth, "LOGIN"):
		a.login = true
		return "LOGIN", nil, nil
	default:
		return "", nil, errors.New("SMTP 服务器未提供支持的认证方式")
	}
	return a.auth.Start(server)
}

func (a *requiredTLSAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if a.auth != nil {
		return a.auth.Next(fromServer, more)
	}
	if !a.login {
		return nil, errors.New("SMTP 认证尚未通过 TLS 检查")
	}
	if !more {
		return nil, nil
	}
	switch string(fromServer) {
	case "Username:":
		return []byte(a.username), nil
	case "Password:":
		return []byte(a.password), nil
	default:
		return nil, errors.New("SMTP LOGIN 认证收到未知质询")
	}
}
