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

	ctx, cancel := context.WithCancel(context.Background())

	bCh, eCh := provider.Provide(ctx, 1, []event.Type{event.AccountCreateEventType, event.VoteEventType, event.UnknownEventType})

	for {
		select {
		case e := <-eCh:
			t.Fatal(e)
		case <-ctx.Done():
			return
		case <-time.Tick(5 * time.Second):
			t.Fatalf("no events within 5 seconds")
		case b := <-bCh:
			require.NotEmpty(t, b.Events)
			cancel()
		}
	}
}

func TestProvider_GenesisBlock(t *testing.T) {
	provider := NewProvider(nodeHTTP, SyncInterval(time.Second))

	ctx, cancel := context.WithCancel(context.Background())

	bCh, eCh := provider.Provide(ctx, 0, []event.Type{event.AccountCreateEventType, event.UnknownEventType})
	for {
		select {
		case e := <-eCh:
			t.Fatal(e)
		case <-ctx.Done():
			return
		case <-time.Tick(1 * time.Minute):
			t.Fatalf("no events within 1 minute")
		case b := <-bCh:
			require.EqualValues(t, b.BlockNum, 0)
			require.NotEmpty(t, b.Events)
			for _, e := range b.Events {
				require.EqualValues(t, event.AccountCreateEventType, e.Type())
			}
			cancel()
		}
	}
}
