package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/scorum/event-provider-go/event"
	"github.com/scorum/event-provider-go/provider"
	scorumhttp "github.com/scorum/scorum-go/transport/http"
)

func main() {
	transport := scorumhttp.NewTransport("https://testnet.scorum.work")
	provider := provider.NewProvider(
		transport,
		provider.WithSyncInterval(time.Second),
		provider.WithBlocksHistoryMaxLimit(100),
		provider.WithRetryTimeout(10*time.Second),
		provider.WithRetryLimit(3),
	)

	ctx, cancel := context.WithCancel(context.Background())

	blocksCh, irreversibleBlocksCh, errorCh := provider.Provide(ctx, 2220447, 2220447,
		[]event.Type{event.CommentEventType, event.PostEventType, event.VoteEventType, event.FlagEventType})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-sigCh
		cancel()
	}()

	for {
		select {
		case e := <-errorCh:
			panic(e)
		case <-ctx.Done():
			return
		case b := <-blocksCh:
			log.Infof("reversible block %d with %d operations", b.BlockNum, len(b.Events))
		case b := <-irreversibleBlocksCh:
			log.Infof("irreversible block %d with %d operations", b.BlockNum, len(b.Events))
		}
	}
}
