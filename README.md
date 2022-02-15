# scorum/event-provider-go
[![Go Report Card](https://goreportcard.com/badge/github.com/scorum/event-provider-go)](https://goreportcard.com/report/github.com/scorum/event-provider-go)
[![GoDoc](https://godoc.org/github.com/scorum/event-provider-go?status.svg)](https://godoc.org/github.com/scorum/event-provider-go)
[![Build Status](https://github.com/scorum/event-provider-go/actions/workflows/main.yml/badge.svg?event=push)](https://github.com/scorum/event-provider-go/actions)

Golang wrapper under [scorum-go](https://github.com/scorum/scorum-go).


## Usage

### with default options:
```go
transport := scorumhttp.NewTransport("https://testnet.scorum.work")
provider := provider.NewProvider(transport)
```

### with custom options:
```go
transport := scorumhttp.NewTransport("https://testnet.scorum.work")
provider := provider.NewProvider(
    transport,
    provider.WithSyncInterval(time.Second),
    provider.WithBlocksHistoryMaxLimit(100),
    provider.WithRetryTimeout(10*time.Second),
    provider.WithRetryLimit(3),
)
```

### without retry:
```go
transport := scorumhttp.NewTransport("https://testnet.scorum.work")
provider := provider.NewProvider(transport, provider.WithOutRetry())
```


## Example

```go
import (
	"context"
	"time"

	"os"
	"os/signal"
	"syscall"

	scorumhttp "github.com/scorum/scorum-go/http"
	"github.com/scorum/event-provider-go/event"
	"github.com/scorum/event-provider-go/provider"
	log "github.com/sirupsen/logrus"
)

func main() {
	transport := scorumhttp.NewTransport("https://testnet.scorum.work")
	provider := provider.NewProvider(transport, provider.SyncInterval(time.Second))

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
```
