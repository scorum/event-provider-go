package provider

import (
	"context"
	"testing"
	"time"

	"github.com/scorum/event-provider-go/event"
	"github.com/stretchr/testify/require"
)

const nodeHTTP = "https://testnet.scorum.com"

func TestProvider(t *testing.T) {
	provider := NewProvider(nodeHTTP, SyncInterval(time.Second))
	done := make(chan bool)

	provider.Provide(context.Background(), 1,
		[]event.Type{event.AccountCreateEventType, event.VoteEventType, event.UnknownEventType},
		func(e event.Event, err error) {
			if err != nil {
				require.NoError(t, err)
			}
			t.Log("event", e.Type())
			done <- true
		})

	select {
	case <-time.Tick(5 * time.Second):
		t.Fatalf("no events within 5 seconds")
	case <-done:
		return
	}
}

func TestProvider_GenesisBlock(t *testing.T) {
	provider := NewProvider(nodeHTTP, SyncInterval(time.Second))
	done := make(chan bool)

	provider.Provide(context.Background(), 0,
		[]event.Type{event.AccountCreateEventType, event.UnknownEventType},
		func(e event.Event, err error) {
			if err != nil {
				require.NoError(t, err)
			}

			if e.Common().BlockNum != 0 {
				done <- true
				return
			}

			if e.Type() == event.AccountCreateEventType {
				t.Log("account", e.(*event.AccountCreateEvent).Account)
			}
		})

	select {
	case <-time.Tick(1 * time.Minute):
		t.Fail()
	case <-done:
		return
	}
}
