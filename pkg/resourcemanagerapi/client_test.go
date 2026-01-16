package resourcemanagerapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTransferPrimary(t *testing.T) {
	expectedTransferee := "rm-1"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if r.URL.Path != "/resource-manager/api/v1/primary/transfer" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var req Primary
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if req.NewPrimary != expectedTransferee {
			t.Fatalf("unexpected new_primary %q", req.NewPrimary)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	defer srv.Close()

	cli := NewClient(srv.URL, 2*time.Second, nil)
	if err := cli.TransferPrimary(context.Background(), expectedTransferee); err != nil {
		t.Fatalf("transfer primary: %v", err)
	}
}

func TestTransferPrimaryFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	cli := NewClient(srv.URL, 2*time.Second, nil)
	if err := cli.TransferPrimary(context.Background(), "rm-1"); err == nil {
		t.Fatalf("expected error")
	}
}
