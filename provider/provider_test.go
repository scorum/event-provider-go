package provider

import (
	"github.com/scorum/event-provider-go/event"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
	"sync"
)

const nodeHTTP = "https://testnet.scorum.com"

func TestProvider(t *testing.T) {
	provider := NewProvider(nodeHTTP, SyncInterval(time.Second))
	wg := sync.WaitGroup{}
	wg.Add(1)

	provider.Provide(1,
		[]event.Type{event.AccountCreateEventType, event.VoteEventType, event.UnknownEventType},
		func(e event.Event, err error) {
			if err != nil {
				require.NoError(t, err)
			}
			t.Log("event", e.Type())
			wg.Done()
		})

	wg.Wait()
}

func TestProvider_GenesisBlock(t *testing.T)  {
	provider := NewProvider(nodeHTTP, SyncInterval(time.Second))
	wg := sync.WaitGroup{}
	wg.Add(1)

	provider.Provide(0,
		[]event.Type{event.AccountCreateEventType},
		func(e event.Event, err error) {
			if err != nil {
				require.NoError(t, err)
			}

			if e.Common().BlockNum != 0 {
				wg.Done()
				return
			}

			t.Log("account", e.(*event.AccountCreateEvent).Account)
		})
	wg.Wait()
}
