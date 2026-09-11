//nolint:testpackage
package dice

import (
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"testing"
	"time"
)

func TestNewMailDialer(t *testing.T) {
	tests := []struct {
		address string
		host    string
		port    int
		ssl     bool
	}{
		{"smtp.qq.com", "smtp.qq.com", 465, true},
		{" smtp.qq.com:465 \n", "smtp.qq.com", 465, true},
		{"smtp.qq.com:587", "smtp.qq.com", 587, false},
		{"smtp.example.com:25", "smtp.example.com", 25, false},
		{"183.47.120.204:465", "183.47.120.204", 465, true},
		{"127.0.0.1:2525", "127.0.0.1", 2525, false},
		{"localhost:1", "localhost", 1, false},
		{"localhost:65535", "localhost", 65535, false},
		{"[::1]:465", "[::1]", 465, true},
	}
	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			dialer, err := newMailDialer(tt.address, "sender@example.com", "test-password")
			if err != nil {
				t.Fatal(err)
			}
			if dialer.Host != tt.host || dialer.Port != tt.port || dialer.SSL != tt.ssl {
				t.Errorf("got host=%q port=%d SSL=%v; want host=%q port=%d SSL=%v",
					dialer.Host, dialer.Port, dialer.SSL, tt.host, tt.port, tt.ssl)
			}
			if dialer.Username != "sender@example.com" || dialer.Password != "test-password" {
				t.Error("SMTP credentials were not preserved")
			}
			if tt.host == "[::1]" && (dialer.TLSConfig == nil || dialer.TLSConfig.ServerName != "::1") {
				t.Error("IPv6 TLS server name must not contain brackets")
			}
		})
	}
}

func TestNewMailDialerRejectsInvalidAddress(t *testing.T) {
	for _, address := range []string{
		"", " \t", ":465", "smtp.qq.com:", "smtp.qq.com:0", "smtp.qq.com:65536",
		"smtp.qq.com:-1", "smtp.qq.com:+465", "smtp.qq.com:smtp", "smtp.qq.com:999999999999999999999",
		"smtp.qq.com:465:25", "183.47.120.204:465:25", "smtp://smtp.qq.com:465",
		"smtp.qq.com/path", "user@smtp.qq.com", "smtp.qq.com?query", "smtp.qq.com#fragment",
		"smtp.qq.com :465", "smtp.\nqq.com", "[invalid::ip]:465",
	} {
		t.Run(address, func(t *testing.T) {
			if _, err := newMailDialer(address, "", ""); err == nil {
				t.Errorf("expected invalid SMTP address %q to be rejected", address)
			}
		})
	}
}

func startRejectingSMTPServer(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	t.Cleanup(func() {
		_ = listener.Close()
		if err := <-done; err != nil {
			t.Errorf("test SMTP server: %v", err)
		}
	})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			done <- acceptErr
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		_, writeErr := fmt.Fprint(conn, "554 test SMTP service unavailable\r\n")
		done <- writeErr
	}()
	return listener.Addr().String()
}

func TestNewMailDialerConnectsToConfiguredPort(t *testing.T) {
	dialer, err := newMailDialer(startRejectingSMTPServer(t), "", "")
	if err != nil {
		t.Fatal(err)
	}
	sender, err := dialer.Dial()
	if sender != nil {
		_ = sender.Close()
	}
	var smtpErr *textproto.Error
	if !errors.As(err, &smtpErr) || smtpErr.Code != 554 {
		t.Fatalf("expected SMTP 554 error from configured port, got %v", err)
	}
}
