package siyi_test

import (
	"context"
	"testing"
	"time"

	"github.com/hitraa/siyi-sdk-go"
	"github.com/hitraa/siyi-sdk-go/pkg/transport"
)

func TestRootPackage_NewClient(t *testing.T) {
	mock := transport.NewMockTransport()
	ctx := context.Background()
	if err := mock.Connect(ctx); err != nil {
		t.Fatalf("mock connect: %v", err)
	}

	cli := siyi.NewClient(
		mock,
		siyi.WithTimeout(500*time.Millisecond),
		siyi.WithRetries(1),
	)

	if cli == nil {
		t.Fatal("expected non-nil client")
	}

	if err := cli.Close(); err != nil {
		t.Errorf("cli close: %v", err)
	}
}
