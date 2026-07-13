//go:build integration

package integration

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNatsPresence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	natsBroker, err := transport.NewNatsBroker(ctx, slog.Default(), natsUrl)
	require.NoError(t, err)
	defer natsBroker.Close()

	subject := "test.presence.livetail"
	timeout := 1 * time.Second
	interval := 300 * time.Millisecond

	isActive, err := natsBroker.StartPresenceListener(ctx, subject, timeout)
	require.NoError(t, err)

	assert.False(t, isActive(), "expected presence to be false initially but got true")

	pubCtx, pubCancel := context.WithCancel(ctx)
	natsBroker.StartPresencePublisher(pubCtx, subject, interval)

	assert.Eventually(t, isActive, 2*time.Second, 10*time.Millisecond,
		"expected presence to become true after publishing")

	pubCancel()

	assert.Eventually(t, func() bool { return !isActive() }, 2*time.Second, 10*time.Millisecond,
		"expected presence to be false after publisher stopped but got true")
}
