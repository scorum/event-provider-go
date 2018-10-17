package provider

import (
	"context"
	"testing"
	"time"

	"github.com/scorum/event-provider-go/event"
	"github.com/scorum/scorum-go"
	"github.com/scorum/scorum-go/sign"
	"github.com/scorum/scorum-go/transport/http"
	"github.com/scorum/scorum-go/types"
	"github.com/stretchr/testify/require"
)

const (
	nodeHTTP = "https://testnet.scorum.com"
	account  = "roselle"
	wif      = "5JwWJ2m2jGG9RPcpDix5AvkDzQZJoZvpUQScsDzzXWAKMs8Q6jH"
)

func TestProvider(t *testing.T) {
	provider := NewProvider(nodeHTTP, SyncInterval(time.Second))

	ctx, cancel := context.WithCancel(context.Background())

	bCh, ibCh, eCh := provider.Provide(ctx, 1, 1, []event.Type{event.AccountCreateEventType, event.VoteEventType, event.UnknownEventType})

	blockRetrieved := false
	irreversibleBlockRetrieved := false

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
			blockRetrieved = true
		case b := <-ibCh:
			require.NotEmpty(t, b.Events)
			irreversibleBlockRetrieved = true
		}

		if blockRetrieved && irreversibleBlockRetrieved {
			cancel()
		}
	}
}

func TestProvider_GenesisBlock(t *testing.T) {
	provider := NewProvider(nodeHTTP, SyncInterval(time.Second))

	ctx, cancel := context.WithCancel(context.Background())

	bCh, ibCh, eCh := provider.Provide(ctx, 0, 0, []event.Type{event.AccountCreateEventType, event.UnknownEventType})

	blockRetrieved := false
	irreversibleBlockRetrieved := false

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
			blockRetrieved = true
		case b := <-ibCh:
			require.EqualValues(t, b.BlockNum, 0)
			require.NotEmpty(t, b.Events)
			for _, e := range b.Events {
				require.EqualValues(t, event.AccountCreateEventType, e.Type())
			}
			irreversibleBlockRetrieved = true
		}

		if blockRetrieved && irreversibleBlockRetrieved {
			cancel()
		}
	}
}

func TestProvider_OnlyIrreversibleBlocksOption(t *testing.T) {
	transport := http.NewTransport(nodeHTTP)
	client := scorumgo.NewClient(transport)

	properties, err := client.Chain.GetChainProperties()
	require.NoError(t, err)

	testOp := &types.VoteOperation{
		Voter:    account,
		Author:   "gina",
		Permlink: "scorum-one-more-post",
		Weight:   0,
	}

	provider := NewProvider(nodeHTTP, SyncInterval(time.Second))

	ctx, cancel := context.WithCancel(context.Background())

	bCh, ibCh, eCh := provider.Provide(ctx, properties.HeadBlockNumber, properties.LastIrreversibleBlockNumber,
		[]event.Type{event.AccountCreateEventType, event.VoteEventType, event.UnknownEventType})

	resp, err := client.Broadcast(sign.TestChain, []string{wif}, testOp)
	require.NoError(t, err)
	blockNum := resp.BlockNum

	blockRetrieved := false
	irreversibleBlockRetrieved := false

	for {
		select {
		case e := <-eCh:
			t.Fatal(e)
		case <-ctx.Done():
			return
		case <-time.Tick(5 * time.Minute):
			t.Fatalf("failed by timeout")
		case b := <-bCh:
			require.NotEmpty(t, b.Events)
			require.NotEmpty(t, b.Events)
			require.True(t, b.BlockNum > properties.HeadBlockNumber)
			require.EqualValues(t, blockNum, b.BlockNum)
			blockRetrieved = true
		case b := <-ibCh:
			require.NotEmpty(t, b.Events)
			require.True(t, b.BlockNum > properties.LastIrreversibleBlockNumber)
			require.True(t, b.BlockNum < properties.LastIrreversibleBlockNumber+21)
			require.Equal(t, blockNum, b.BlockNum)
			irreversibleBlockRetrieved = true
		}

		if blockRetrieved && irreversibleBlockRetrieved {
			cancel()
		}
	}
}
