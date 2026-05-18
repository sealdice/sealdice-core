package realtime

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestEncodeEnvelopeIncludesEventNameAndPayload(t *testing.T) {
	data, err := encodeEnvelope(Event{
		Name:    EventLogsAppend,
		Payload: LogAppendPayload{Item: nil},
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
	if eventName != EventLogsAppend {
		t.Fatalf("event = %q, want %q", eventName, EventLogsAppend)
	}
	if _, ok := decoded["payload"]; !ok {
		t.Fatal("payload field missing")
	}
}

func TestWriteSSEEventFormatsStreamFrame(t *testing.T) {
	var buf bytes.Buffer

	err := writeSSEEvent(&buf, Event{
		Name:    EventSystemReady,
		Payload: SystemReadyPayload{},
	})
	if err != nil {
		t.Fatalf("writeSSEEvent returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "event: "+EventSystemReady+"\n") {
		t.Fatalf("event line missing: %q", output)
	}
	if !strings.Contains(output, "data: ") {
		t.Fatalf("data line missing: %q", output)
	}
	if !strings.HasSuffix(output, "\n\n") {
		t.Fatalf("sse frame should end with blank line: %q", output)
	}
}
