//nolint:testpackage // Tests intentionally cover internal error-wrapping helpers.
package emitter

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
)

func TestResponseUnmarshalReadsStandardRetcode(t *testing.T) {
	var resp Response[sonic.NoCopyRawMessage]
	if err := sonic.Unmarshal([]byte(`{"status":"failed","retcode":100,"data":{"message":"boom"},"echo":"echo-1"}`), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Status != "failed" {
		t.Fatalf("status = %q, want failed", resp.Status)
	}
	if resp.RetCode != 100 {
		t.Fatalf("retcode = %d, want 100", resp.RetCode)
	}
	if string(resp.Data) != `{"message":"boom"}` {
		t.Fatalf("data = %s, want payload preserved", resp.Data)
	}
	if resp.Echo != "echo-1" {
		t.Fatalf("echo = %q, want echo-1", resp.Echo)
	}
}

func TestWrapActionErrorTimeoutIncludesChineseContext(t *testing.T) {
	err := wrapActionError(context.DeadlineExceeded, ACTION_SET_FRIEND_ADD_REQUEST, "echo-timeout", map[string]any{
		"flag":    "friend-flag",
		"approve": true,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "等待 OneBot Echo 超时") {
		t.Fatalf("expected timeout message in %q", msg)
	}
	if !strings.Contains(msg, "set_friend_add_request") {
		t.Fatalf("expected action in %q", msg)
	}
	if !strings.Contains(msg, "echo-timeout") {
		t.Fatalf("expected echo id in %q", msg)
	}
	if !strings.Contains(msg, `"flag":"friend-flag"`) {
		t.Fatalf("expected params in %q", msg)
	}
}

func TestDecodeResponseFailedStatusIncludesRequestAndResponse(t *testing.T) {
	resp := Response[sonic.NoCopyRawMessage]{
		Status:  "failed",
		RetCode: 1401,
		Data:    sonic.NoCopyRawMessage(`{"message":"denied"}`),
		Echo:    "echo-failed",
	}

	_, err := decodeResponse[map[string]any](ACTION_SET_GROUP_ADD_REQUEST, "echo-failed", map[string]any{
		"flag":     "group-flag",
		"sub_type": "invite",
		"approve":  false,
	}, resp)
	if err == nil {
		t.Fatal("expected failed-status error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "OneBot 动作执行失败") {
		t.Fatalf("expected failed-status message in %q", msg)
	}
	if !strings.Contains(msg, `"sub_type":"invite"`) {
		t.Fatalf("expected request params in %q", msg)
	}
	if !strings.Contains(msg, `"retcode":1401`) {
		t.Fatalf("expected response retcode in %q", msg)
	}
	if !strings.Contains(msg, `"message":"denied"`) {
		t.Fatalf("expected response data in %q", msg)
	}
}

func TestDecodeResponseDecodeFailureIncludesRawResponse(t *testing.T) {
	resp := Response[sonic.NoCopyRawMessage]{
		Status:  "ok",
		RetCode: 0,
		Data:    sonic.NoCopyRawMessage(`{"group_id":"bad-int"}`),
		Echo:    "echo-decode",
	}

	_, err := decodeResponse[struct {
		GroupID int64 `json:"group_id"`
	}](ACTION_GET_GROUP_INFO, "echo-decode", map[string]any{
		"group_id": 12345,
		"no_cache": true,
	}, resp)
	if err == nil {
		t.Fatal("expected decode error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "解析 OneBot 响应失败") {
		t.Fatalf("expected decode message in %q", msg)
	}
	if !strings.Contains(msg, `"group_id":"bad-int"`) {
		t.Fatalf("expected raw response in %q", msg)
	}
	if !strings.Contains(msg, `"group_id":12345`) {
		t.Fatalf("expected request params in %q", msg)
	}
}

func TestWaitEchoAfterSendTimeoutWrapsActionContext(t *testing.T) {
	oldTimeout := EchoTimeOut
	EchoTimeOut = 10 * time.Millisecond
	defer func() {
		EchoTimeOut = oldTimeout
	}()

	e := &emitterSocket{}
	_, err := e.waitEchoAfterSend(t.Context(), ACTION_GET_GROUP_INFO, "echo-timeout-2", map[string]any{
		"group_id": 12345,
	}, func() error {
		return nil
	})
	if err == nil {
		t.Fatal("expected timeout")
	}

	msg := err.Error()
	if !strings.Contains(msg, "等待 OneBot Echo 超时") {
		t.Fatalf("expected timeout message in %q", msg)
	}
	if !strings.Contains(msg, "get_group_info") {
		t.Fatalf("expected action in %q", msg)
	}
}

func TestWrapActionErrorIncludesOriginalFailure(t *testing.T) {
	err := wrapActionError(errors.New("write failed"), ACTION_SET_GROUP_ADD_REQUEST, "echo-send", map[string]any{
		"flag":     "group-flag",
		"sub_type": "invite",
	})
	if err == nil {
		t.Fatal("expected wrapped error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "write failed") {
		t.Fatalf("expected original error in %q", msg)
	}
	if !strings.Contains(msg, `"sub_type":"invite"`) {
		t.Fatalf("expected params in %q", msg)
	}
}
