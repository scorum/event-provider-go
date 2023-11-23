package provider

import (
	"context"
	"testing"
	"time"

	"github.com/scorum/event-provider-go/event"
	scorumgo "github.com/scorum/scorum-go"
	"github.com/scorum/scorum-go/key"
	"github.com/scorum/scorum-go/rpc"
	"github.com/scorum/scorum-go/sign"

	"github.com/scorum/scorum-go/types"
	"github.com/stretchr/testify/require"
)

const (
	nodeHTTP = "https://testnet.scorum.work"
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
			require.EqualValues(t, 0, b.BlockNum)
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

func TestProvider_Provide(t *testing.T) {
	transport := rpc.NewHTTPTransport(nodeHTTP)
	client := scorumgo.NewClient(transport)

	properties, err := client.Chain.GetChainProperties(context.Background())
	require.NoError(t, err)

	testOp := &types.VoteOperation{
		Voter:    account,
		Author:   "sheldon",
		Permlink: "online-sportsbook-for-usa-residents-20-cashback-on-first-deposit",
		Weight:   0,
	}

	provider := NewProvider(nodeHTTP, SyncInterval(time.Second))

	ctx, cancel := context.WithCancel(context.Background())

	bCh, ibCh, eCh := provider.Provide(ctx, properties.HeadBlockNumber, properties.LastIrreversibleBlockNumber,
		[]event.Type{event.AccountCreateEventType, event.VoteEventType, event.UnknownEventType})

	k, err := key.PrivateKeyFromString(wif)
	require.NoError(t, err)

	resp, err := client.BroadcastTransactionSynchronous(context.TODO(), sign.TestNetChainID, []types.Operation{testOp}, k)
	require.NoError(t, err)
	blockNum := resp.BlockNum

	headBlockNumber := properties.HeadBlockNumber
	lastIrreversibleBlockNumber := properties.LastIrreversibleBlockNumber

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
			if blockRetrieved {
				continue
			}

			require.False(t, b.BlockNum > blockNum)
			if b.BlockNum == blockNum {
				require.NotEmpty(t, b.Events)
				require.NotEmpty(t, b.Events)
				require.True(t, b.BlockNum > headBlockNumber)
				require.EqualValues(t, blockNum, b.BlockNum)
				blockRetrieved = true
			}
		case b := <-ibCh:
			if irreversibleBlockRetrieved {
				continue
			}

			require.False(t, b.BlockNum > blockNum)
			if b.BlockNum == blockNum {
				require.NotEmpty(t, b.Events)
				require.True(t, b.BlockNum > lastIrreversibleBlockNumber)

				properties, err := client.Chain.GetChainProperties(context.Background())
				require.NoError(t, err)

				require.True(t, b.BlockNum <= properties.LastIrreversibleBlockNumber)
				require.Equal(t, blockNum, b.BlockNum)
				irreversibleBlockRetrieved = true
			}
		}

		if blockRetrieved && irreversibleBlockRetrieved {
			cancel()
		}
	}
}
