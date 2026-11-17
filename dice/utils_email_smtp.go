package dice

import (
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/gomail.v2"
)

// newMailDialer accepts a hostname or host:port, defaulting to implicit TLS on 465.
// Port 587 requires STARTTLS; other explicit ports retain gomail's STARTTLS policy.
func newMailDialer(address, username, password string) (*gomail.Dialer, error) {
	address = strings.TrimSpace(address)
	host, port := address, 465
	if strings.Contains(address, ":") {
		var portText string
		var err error
		host, portText, err = net.SplitHostPort(address)
		if err != nil {
			return nil, errors.New("SMTP 地址格式错误，请填写主机名或主机名:端口，例如 smtp.qq.com:465")
		}
		if portText == "" || strings.IndexFunc(portText, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			return nil, errors.New("SMTP 端口必须是 1 到 65535 之间的整数")
		}
		port, err = strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return nil, errors.New("SMTP 端口必须是 1 到 65535 之间的整数")
		}
	}
	if host == "" || strings.ContainsAny(host, "/\\@?#[]") || strings.IndexFunc(host, unicode.IsSpace) >= 0 {
		return nil, errors.New("SMTP 主机名无效，请填写主机名，不要包含协议前缀或路径")
	}
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return nil, errors.New("SMTP IPv6 地址无效")
	}

	dialer := gomail.NewDialer(host, port, username, password)
	if strings.Contains(host, ":") {
		// gomail concatenates Host and Port instead of using net.JoinHostPort.
		dialer.Host = "[" + host + "]"
		dialer.TLSConfig = &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	}
	if port == 587 {
		// A non-nil Auth is always called by gomail, even without an AUTH capability.
		// Its TLS check runs before any AUTH command or message can be sent, including
		// when gomail reconnects. Do not allow its opportunistic STARTTLS fallback.
		dialer.Auth = &requiredTLSAuth{host: dialer.Host, username: username, password: password}
	}
	return dialer, nil
}
