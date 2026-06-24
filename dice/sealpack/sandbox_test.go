package sealpack_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"sealdice-core/dice/sealpack"
)

func TestSandboxedHTTPPostPreservesRawBody(t *testing.T) {
	want := []byte{0x00, 0xff, 0x61, 0xc3, 0x28}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !bytes.Equal(got, want) {
			http.Error(w, "body mismatch", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	sandbox := sealpack.NewSandbox("tester/package", &sealpack.Permissions{Network: true}, "", "")
	client := sealpack.NewSandboxedHTTP(sandbox)
	resp, err := client.Post(server.URL, "application/octet-stream", want)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Post() status = %d, body = %s", resp.StatusCode, string(body))
	}
}
