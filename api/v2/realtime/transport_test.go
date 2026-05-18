package realtime_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	realtime "sealdice-core/api/v2/realtime"
)

func TestEncodeEnvelopeIncludesEventNameAndPayload(t *testing.T) {
	data, err := realtime.EncodeEnvelope(realtime.Event{
		Name:    realtime.EventLogsAppend,
		Payload: realtime.LogAppendPayload{Item: nil},
	})
	if err != nil {
		t.Fatalf("encodeEnvelope returned error: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	var eventName string
	if err := json.Unmarshal(decoded["event"], &eventName); err != nil {
		t.Fatalf("event field invalid: %v", err)
	}
	if eventName != realtime.EventLogsAppend {
		t.Fatalf("event = %q, want %q", eventName, realtime.EventLogsAppend)
	}
	if _, ok := decoded["payload"]; !ok {
		t.Fatal("payload field missing")
	}
}

func TestWriteSSEEventFormatsStreamFrame(t *testing.T) {
	var buf bytes.Buffer

	err := realtime.WriteSSEEvent(&buf, realtime.Event{
		Name:    realtime.EventSystemReady,
		Payload: realtime.SystemReadyPayload{},
	})
	if err != nil {
		t.Fatalf("writeSSEEvent returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "event: "+realtime.EventSystemReady+"\n") {
		t.Fatalf("event line missing: %q", output)
	}
	if !strings.Contains(output, "data: ") {
		t.Fatalf("data line missing: %q", output)
	}
	if !strings.HasSuffix(output, "\n\n") {
		t.Fatalf("sse frame should end with blank line: %q", output)
	}
}
