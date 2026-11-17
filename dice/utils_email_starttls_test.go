//nolint:testpackage // 验证包内的 SMTP 拨号器及强制 TLS 认证。
package dice

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/gomail.v2"
)

const (
	smtpTestUsername = "sender@example.com"
	smtpTestPassword = "test-smtp-secret"
	smtpTestBody     = "private test message"
)

type submissionTestServer struct {
	startTLS  bool
	rejectTLS bool
	mechanism string
	tlsConfig *tls.Config
}

type submissionTestResult struct {
	plaintextCommands []string
	authenticated     bool
	body              string
	err               error
}

// serve accepts one connection and records everything sent before TLS.
func (s submissionTestServer) serve(listener net.Listener) (result submissionTestResult) {
	conn, err := listener.Accept()
	if err != nil {
		result.err = err
		return result
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	wire := textproto.NewConn(conn)
	if err = wire.PrintfLine("220 test SMTP ready"); err != nil {
		result.err = err
		return result
	}
	encrypted := false
	for {
		line, readErr := wire.ReadLine()
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				result.err = readErr
			}
			return result
		}
		if !encrypted {
			result.plaintextCommands = append(result.plaintextCommands, line)
		}
		switch {
		case strings.HasPrefix(line, "EHLO "):
			response := "250-test SMTP\r\n"
			if s.startTLS && !encrypted {
				response += "250-STARTTLS\r\n"
			}
			if s.mechanism != "" {
				response += "250-AUTH " + s.mechanism + "\r\n"
			}
			err = wire.PrintfLine("%s250 OK", response)
		case line == "STARTTLS":
			if s.rejectTLS {
				err = wire.PrintfLine("454 TLS unavailable")
				break
			}
			if err = wire.PrintfLine("220 begin TLS"); err != nil {
				break
			}
			secured := tls.Server(conn, s.tlsConfig)
			if err = secured.Handshake(); err != nil {
				// Certificate rejection is expected in the negative TLS cases.
				return result
			}
			wire = textproto.NewConn(secured)
			encrypted = true
		case strings.HasPrefix(line, "AUTH "):
			err = s.authenticate(wire, line)
			if err == nil {
				result.authenticated = true
				err = wire.PrintfLine("235 authenticated")
			}
		case strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
			if !result.authenticated {
				err = errors.New("message envelope sent without authentication")
				break
			}
			err = wire.PrintfLine("250 OK")
		case line == "DATA":
			if err = wire.PrintfLine("354 send message"); err == nil {
				var body []byte
				body, err = wire.ReadDotBytes()
				result.body = string(body)
				if err == nil {
					err = wire.PrintfLine("250 accepted")
				}
			}
		case line == "QUIT":
			result.err = wire.PrintfLine("221 bye")
			return result
		default:
			err = fmt.Errorf("unexpected SMTP command: %q", line)
		}
		if err != nil {
			result.err = err
			return result
		}
	}
}

func (s submissionTestServer) authenticate(wire *textproto.Conn, line string) error {
	encode := base64.StdEncoding.EncodeToString
	switch s.mechanism {
	case "PLAIN":
		if line != "AUTH PLAIN "+encode([]byte("\x00"+smtpTestUsername+"\x00"+smtpTestPassword)) {
			return errors.New("incorrect PLAIN credentials")
		}
	case "LOGIN", "CRAM-MD5":
		if line != "AUTH "+s.mechanism {
			return errors.New("incorrect AUTH mechanism")
		}
		challenges := []string{"Username:", "Password:"}
		responses := []string{smtpTestUsername, smtpTestPassword}
		if s.mechanism == "CRAM-MD5" {
			challenges = []string{"test-smtp-challenge"}
			auth := smtp.CRAMMD5Auth(smtpTestUsername, smtpTestPassword)
			response, err := auth.Next([]byte(challenges[0]), true)
			if err != nil {
				return err
			}
			responses = []string{string(response)}
		}
		for i, challenge := range challenges {
			if err := wire.PrintfLine("334 %s", encode([]byte(challenge))); err != nil {
				return err
			}
			response, err := wire.ReadLine()
			if err != nil {
				return err
			}
			if response != encode([]byte(responses[i])) {
				return errors.New("incorrect challenge response")
			}
		}
	default:
		return errors.New("AUTH sent without an advertised mechanism")
	}
	return nil
}

func TestSubmissionRequiresSTARTTLS(t *testing.T) {
	certServer := httptest.NewTLSServer(nil)
	defer certServer.Close()
	roots := x509.NewCertPool()
	roots.AddCert(certServer.Certificate())

	tests := []struct {
		name          string
		mechanism     string
		startTLS      bool
		rejectTLS     bool
		untrustedCert bool
		wrongHostname bool
		wantSuccess   bool
	}{
		{name: "LOGIN without STARTTLS", mechanism: "LOGIN"},
		{name: "PLAIN without STARTTLS", mechanism: "PLAIN"},
		{name: "CRAM-MD5 without STARTTLS", mechanism: "CRAM-MD5"},
		{name: "no STARTTLS or AUTH"},
		{name: "STARTTLS command rejected", mechanism: "LOGIN", startTLS: true, rejectTLS: true},
		{name: "untrusted certificate", mechanism: "LOGIN", startTLS: true, untrustedCert: true},
		{name: "wrong certificate hostname", mechanism: "LOGIN", startTLS: true, wrongHostname: true},
		{name: "TLS without AUTH", startTLS: true},
		{name: "TLS with PLAIN", mechanism: "PLAIN", startTLS: true, wantSuccess: true},
		{name: "TLS with LOGIN", mechanism: "LOGIN", startTLS: true, wantSuccess: true},
		{name: "TLS with CRAM-MD5", mechanism: "CRAM-MD5", startTLS: true, wantSuccess: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			server := submissionTestServer{
				startTLS: tt.startTLS, rejectTLS: tt.rejectTLS,
				mechanism: tt.mechanism, tlsConfig: certServer.TLS,
			}
			done := make(chan submissionTestResult, 1)
			go func() { done <- server.serve(listener) }()

			dialer, err := newMailDialer("127.0.0.1:587", smtpTestUsername, smtpTestPassword)
			if err != nil {
				t.Fatal(err)
			}
			// Keep the policy selected for 587, using an ephemeral local test port.
			_, portText, err := net.SplitHostPort(listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			dialer.Port, err = strconv.Atoi(portText)
			if err != nil {
				t.Fatal(err)
			}
			dialer.TLSConfig = &tls.Config{RootCAs: roots, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12}
			if tt.untrustedCert {
				dialer.TLSConfig.RootCAs = x509.NewCertPool()
			}
			if tt.wrongHostname {
				dialer.TLSConfig.ServerName = "wrong.example"
			}
			message := gomail.NewMessage()
			message.SetHeader("From", smtpTestUsername)
			message.SetHeader("To", "recipient@example.com")
			message.SetBody("text/plain", smtpTestBody)
			err = dialer.DialAndSend(message)
			if (err == nil) != tt.wantSuccess {
				t.Errorf("DialAndSend error = %v, want success = %v", err, tt.wantSuccess)
			}
			if !tt.startTLS && (err == nil || !strings.Contains(err.Error(), "必须使用 STARTTLS")) {
				t.Errorf("expected a mandatory STARTTLS error, got %v", err)
			}
			result := <-done
			if result.err != nil {
				t.Fatal(result.err)
			}
			for _, command := range result.plaintextCommands {
				if !strings.HasPrefix(command, "EHLO ") && command != "STARTTLS" && command != "QUIT" {
					t.Errorf("unexpected plaintext command: %q", command)
				}
			}
			if tt.wantSuccess {
				if !result.authenticated || !strings.Contains(result.body, smtpTestBody) {
					t.Error("expected authenticated delivery over TLS")
				}
			} else if result.authenticated || result.body != "" {
				t.Error("authentication or delivery occurred despite missing/failed TLS or AUTH")
			}
		})
	}
}

func TestRequiredTLSAuthRejectsDowngradeOnReuse(t *testing.T) {
	for _, mechanism := range []string{"PLAIN", "LOGIN", "CRAM-MD5"} {
		t.Run(mechanism, func(t *testing.T) {
			auth := &requiredTLSAuth{host: "smtp.example.com", username: smtpTestUsername, password: smtpTestPassword}
			server := &smtp.ServerInfo{Name: auth.host, TLS: true, Auth: []string{mechanism}}
			if _, _, err := auth.Start(server); err != nil {
				t.Fatal(err)
			}
			server.TLS = false
			if _, data, err := auth.Start(server); err == nil || len(data) != 0 {
				t.Fatal("reused authentication must reject an unencrypted connection without returning credentials")
			}
			if data, err := auth.Next([]byte("Password:"), true); err == nil || len(data) != 0 {
				t.Fatal("failed TLS check must clear the previous authentication state")
			}
		})
	}
}
