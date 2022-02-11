package provider

import (
	"context"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/scorum/event-provider-go/event"
	scorumgo "github.com/scorum/scorum-go"
	"github.com/scorum/scorum-go/caller"
	"github.com/scorum/scorum-go/retrycaller"
)

const (
	timeLayout = "2006-01-02T15:04:05"
)

type Option func(*Provider)

func WithSyncInterval(interval time.Duration) Option {
	return func(p *Provider) {
		p.syncInterval = interval
	}
}

func WithBlocksHistoryMaxLimit(limit uint32) Option {
	return func(p *Provider) {
		p.blocksHistoryMaxLimit = limit
	}
}

func WithRetryTimeout(timeout time.Duration) Option {
	return func(p *Provider) {
		p.retryTimeout = timeout
	}
}

func WithRetryLimit(limit int) Option {
	return func(p *Provider) {
		p.retryLimit = limit
	}
}

func WithOutRetry() Option {
	return func(p *Provider) {
		p.retryLimit = 0
		p.retryTimeout = 0
	}
}

func withRetry(transport caller.CallCloser, timeout time.Duration, limit int) caller.CallCloser {
	if timeout == 0 && limit == 0 {
		return transport
	}

	retryOptions := retrycaller.RetryOptions{
		Timeout:    timeout,
		RetryLimit: limit,
	}
	return retrycaller.NewRetryCaller(transport, retrycaller.WithDefaultRetry(retryOptions))
}

type Provider struct {
	client                *scorumgo.Client
	syncInterval          time.Duration
	blocksHistoryMaxLimit uint32
	retryTimeout          time.Duration
	retryLimit            int

	// mutable
	CurrentBlockNum uint32
}

// NewProvider construct new provider
//
// with default options:
// 	transport := scorumhttp.NewTransport("https://testnet.scorum.work")
// 	provider := provider.NewProvider(transport)
//
// with custom options:
// 	transport := scorumhttp.NewTransport("https://testnet.scorum.work")
// 	provider := provider.NewProvider(
// 		transport,
// 		provider.WithSyncInterval(time.Second),
// 		provider.WithBlocksHistoryMaxLimit(100),
// 		provider.WithRetryTimeout(10*time.Second),
// 		provider.WithRetryLimit(3),
//	)
//
// without retry:
// 	transport := scorumhttp.NewTransport("https://testnet.scorum.work")
// 	provider := provider.NewProvider(transport, provider.WithOutRetry())
//
func NewProvider(transport caller.CallCloser, options ...Option) *Provider {
	p := Provider{
		syncInterval:          time.Second,
		blocksHistoryMaxLimit: 100,
		retryLimit:            3,
		retryTimeout:          10 * time.Second,
	}

	for _, option := range options {
		option(&p)
	}

	p.client = scorumgo.NewClient(withRetry(transport, p.retryTimeout, p.retryLimit))

	return &p
}

func (p *Provider) Provide(ctx context.Context, from, irreversibleFrom uint32, eventTypes []event.Type) (chan event.Block, chan event.Block, chan error) {
	if irreversibleFrom > from {
		log.Warn("EventProvider: irreversibleFrom > from")
	}

	log.Infof("Provide starting... from : %d; irreversible from: %d", from, irreversibleFrom)

	blocksCh := make(chan event.Block)
	irreversibleBlocksCh := make(chan event.Block)
	errCh := make(chan error)

	go func(blocksCh, irreversibleBlocksCh chan event.Block, errCh chan error) {
		// genesis block

		if from == 0 {

			accountCreateEventTypeFound := false
			for _, eventType := range eventTypes {
				if eventType == event.AccountCreateEventType {
					accountCreateEventTypeFound = true
					break
				}
			}

			if accountCreateEventTypeFound {
				accounts, err := p.getExistingAccounts(ctx)
				if err != nil {
					errCh <- err
					return
				}

				// genesis block
				genesis := event.Block{
					BlockNum:  0,
					Timestamp: time.Unix(0, 0),
				}

				for _, account := range accounts {
					genesis.Events = append(genesis.Events,
						&event.AccountCreateEvent{
							Account: account,
						})

				}
				blocksCh <- genesis
				irreversibleBlocksCh <- genesis
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
				properties, err := p.client.Chain.GetChainProperties(ctx)

				if err != nil {
					errCh <- err
					return
				}

				if from >= properties.HeadBlockNumber {
					time.Sleep(p.syncInterval)
					continue
				}

				// GetBlockHistory has descending order
				limit := properties.HeadBlockNumber - irreversibleFrom
				if limit > p.blocksHistoryMaxLimit {
					limit = p.blocksHistoryMaxLimit
				}

				offset := from + limit

				history, err := p.client.BlockchainHistory.GetBlocks(ctx, offset, limit)
				if err != nil {
					errCh <- err
					return
				}

				nums := make([]uint32, 0, len(history))
				for num := range history {
					nums = append(nums, num)
				}
				sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })

				for _, num := range nums {
					p.CurrentBlockNum = num

					block := history[num]

					timestamp, err := time.Parse(timeLayout, block.Timestamp)
					if err != nil {
						errCh <- err
						return
					}

					eBlock := event.Block{
						BlockNum:  num,
						Timestamp: timestamp,
					}

					for _, operation := range block.Operations {
						ev := event.ToEvent(operation.Operation)
						for _, eventType := range eventTypes {
							if ev.Type() == eventType {
								eBlock.Events = append(eBlock.Events, ev)
								break
							}
						}
					}

					if len(eBlock.Events) != 0 {
						if num > from {
							blocksCh <- eBlock
						}

						if num <= properties.LastIrreversibleBlockNumber && num > irreversibleFrom {
							irreversibleBlocksCh <- eBlock
						}
					}

					if num > from {
						from = num
					}
					if (num <= properties.LastIrreversibleBlockNumber) && (num > irreversibleFrom) {
						irreversibleFrom = num
					}
				}
			}
		}

	}(blocksCh, irreversibleBlocksCh, errCh)

	return blocksCh, irreversibleBlocksCh, errCh
}

func (p *Provider) getExistingAccounts(ctx context.Context) ([]string, error) {
	const lookupAccountsMaxLimit = 1000

	lowerBound := ""
	result := make([]string, 0, lookupAccountsMaxLimit)

	for {
		accounts, err := p.client.Database.LookupAccounts(ctx, lowerBound, lookupAccountsMaxLimit)
		if err != nil {
			return nil, err
		}

		if lowerBound == "" {
			accounts = accounts[:]
		} else {
			accounts = accounts[1:]
		}

		if len(accounts) == 0 {
			break
		}

		lowerBound = accounts[len(accounts)-1]
		result = append(result, accounts...)
	}

	return result, nil
}
