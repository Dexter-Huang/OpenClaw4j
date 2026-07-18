package contextx

import (
	"context"
	"testing"
	"time"
)

func TestWithAndFromReturnsRequestContext(t *testing.T) {
	rc := &RequestContext{
		RequestID:   "req-1",
		AccountID:   "10000",
		Username:    "saa",
		AccountType: "admin",
		WorkspaceID: "1",
		CallerIP:    "127.0.0.1",
		Source:      "console",
		AuthType:    AuthTypeConsoleToken,
		StartTime:   time.Unix(1, 0),
	}

	got, ok := From(With(context.Background(), rc))
	if !ok {
		t.Fatal("expected request context to exist")
	}
	if got.RequestID != "req-1" || got.AccountID != "10000" || got.WorkspaceID != "1" {
		t.Fatalf("unexpected request context: %#v", got)
	}
}

func TestFromMissingContextReturnsFalse(t *testing.T) {
	if got, ok := From(context.Background()); ok || got != nil {
		t.Fatalf("expected missing context, got=%#v ok=%v", got, ok)
	}
}

func TestDetachKeepsRequestContextWithoutParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	rc := &RequestContext{RequestID: "req-2", AccountID: "10000"}
	detached := Detach(With(parent, rc))
	cancel()

	select {
	case <-detached.Done():
		t.Fatal("detached context should not inherit parent cancellation")
	default:
	}

	got, ok := From(detached)
	if !ok || got.RequestID != "req-2" {
		t.Fatalf("unexpected detached request context: %#v ok=%v", got, ok)
	}
}
