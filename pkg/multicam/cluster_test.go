package multicam_test

import (
	"context"
	"testing"
	"time"

	"github.com/hitraa/siyi-sdk-go"
	"github.com/hitraa/siyi-sdk-go/pkg/multicam"
	"github.com/hitraa/siyi-sdk-go/pkg/transport"
)

func TestCluster_MockNodes(t *testing.T) {
	cluster := multicam.NewCluster()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mockPrimary := transport.NewMockTransport(false)
	_ = mockPrimary.Connect(ctx)
	clientPrimary := siyi.NewClient(mockPrimary)
	_ = clientPrimary.Connect(ctx)

	mockSecondary := transport.NewMockTransport(false)
	_ = mockSecondary.Connect(ctx)
	clientSecondary := siyi.NewClient(mockSecondary)
	_ = clientSecondary.Connect(ctx)

	cluster.AddClient("MainCamera", multicam.RolePrimary, clientPrimary)
	cluster.AddClient("CameraB", multicam.RoleSecondary, clientSecondary)

	if cluster.Primary() != clientPrimary {
		t.Errorf("expected Primary client to match")
	}
	if cluster.Secondary() != clientSecondary {
		t.Errorf("expected Secondary client to match")
	}
	if _, ok := cluster.Node("MainCamera"); !ok {
		t.Errorf("expected MainCamera node to exist")
	}

	_ = cluster.SyncCapture(ctx)
	_ = cluster.Close()
}
