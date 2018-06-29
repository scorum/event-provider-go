package provider

import (
	"github.com/scorum/event-provider-go/event"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const nodeHTTP = "https://testnet.scorum.com"

func TestMonitor_Provide(t *testing.T) {
	monitor := NewProvider(nodeHTTP, SyncInterval(time.Second))

	c, e := monitor.Provide(0,
		[]event.Type{event.AccountCreateEventType, event.VoteEventType, event.UnknownEventType}, 100)

	received := false

	for {
		select {
		case <-c:
			received = true
		case err := <-e:
			require.NoError(t, err)
		}

		if received {
			break
		}
	}
}
