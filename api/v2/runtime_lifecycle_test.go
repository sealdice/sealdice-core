package v2_test

import (
	"context"
	"testing"

	"go.uber.org/goleak"

	apiv2 "sealdice-core/api/v2"
	"sealdice-core/api/v2/realtime"
)

func TestOpenAPIRegistrationDoesNotStartWorkers(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	for range 3 {
		apiv2.BuildOpenAPI()
	}
}

func TestRealtimeRuntimeCanBeCancelledAndClosedRepeatedly(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := realtime.NewServer(nil)
	server.Start(ctx)
	server.Start(ctx)
	cancel()
	server.Close()
	server.Close()
	server.Start(context.Background()) // Closed runtimes cannot restart.
}
