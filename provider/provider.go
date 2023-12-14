package provider

import (
	"context"
	"sort"
	"time"

	"github.com/scorum/event-provider-go/event"
	"github.com/scorum/scorum-go"
	"github.com/scorum/scorum-go/apis/blockchain_history"
	"github.com/scorum/scorum-go/apis/chain"
	"github.com/scorum/scorum-go/caller"
	"github.com/scorum/scorum-go/rpc"
	log "github.com/sirupsen/logrus"
)

const (
	timeLayout = "2006-01-02T15:04:05"
)

type Options struct {
	// SyncInterval is an interval to poll the blockchain
	SyncInterval          time.Duration
	BlocksHistoryMaxLimit uint32
	ErrorRetryTimeout     time.Duration
	ErrorRetryLimit       int
	ProvideEmptyBlocks    bool
}

type Option func(*Options)

func SyncInterval(interval time.Duration) Option {
	return func(args *Options) {
		args.SyncInterval = interval
	}
}

func BlocksHistoryMaxLimit(limit uint32) Option {
	return func(args *Options) {
		args.BlocksHistoryMaxLimit = limit
	}
}

func ErrorRetryTimeout(timeout time.Duration) Option {
	return func(args *Options) {
		args.ErrorRetryTimeout = timeout
	}
}

func ErrorRetryLimit(limit int) Option {
	return func(args *Options) {
		args.ErrorRetryLimit = limit
	}
}

func ProvideEmptyBlocks(v bool) Option {
	return func(args *Options) {
		args.ProvideEmptyBlocks = v
	}
}

type Provider struct {
	client          *scorumgo.Client
	Options         *Options
	CurrentBlockNum uint32
}

func NewProviderWithClient(client caller.CallCloser, setters ...Option) *Provider {
	args := &Options{
		SyncInterval:          time.Second,
		BlocksHistoryMaxLimit: 100,
		ErrorRetryTimeout:     10 * time.Second,
		ErrorRetryLimit:       3,
		ProvideEmptyBlocks:    false,
	}

	for _, setter := range setters {
		setter(args)
	}

	return &Provider{
		client:  scorumgo.NewClient(client),
		Options: args,
	}
}

func NewProvider(url string, setters ...Option) *Provider {
	return NewProviderWithClient(rpc.NewHTTPTransport(url), setters...)
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
				properties, err := p.getChainProperties(ctx)

				if err != nil {
					errCh <- err
					return
				}

				if from >= properties.HeadBlockNumber {
					time.Sleep(p.Options.SyncInterval)
					continue
				}

				// GetBlockHistory has descending order
				limit := properties.HeadBlockNumber - irreversibleFrom
				if limit > p.Options.BlocksHistoryMaxLimit {
					limit = p.Options.BlocksHistoryMaxLimit
				}

				offset := from + limit

				history, err := p.getBlockHistory(ctx, offset, limit)
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

					if len(eBlock.Events) != 0 || p.Options.ProvideEmptyBlocks {
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

					time.Sleep(p.Options.SyncInterval)
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
		accounts, err := p.lookupAccounts(ctx, lowerBound, lookupAccountsMaxLimit)
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

func (p *Provider) lookupAccounts(ctx context.Context, lowerBoundName string, limit uint16) (names []string, err error) {
	TryDo(func(attempt int) (retry bool, err error) {
		names, err = p.client.Database.LookupAccounts(ctx, lowerBoundName, limit)
		if err != nil {
			time.Sleep(p.Options.ErrorRetryTimeout)
		}
		return attempt < p.Options.ErrorRetryLimit, err
	})
	return
}

func (p *Provider) getChainProperties(ctx context.Context) (prop *chain.ChainProperties, err error) {
	TryDo(func(attempt int) (retry bool, err error) {
		prop, err = p.client.Chain.GetChainProperties(ctx)

		// log.Debugf("getChainProperties dump: ", spew.Sdump(prop))

		if err != nil {
			log.Debugf("getChainProperties error retry: ")

			time.Sleep(p.Options.ErrorRetryTimeout)
		}
		return attempt < p.Options.ErrorRetryLimit, err
	})
	return
}

func (p *Provider) getBlockHistory(ctx context.Context, blockNum, limit uint32) (history blockchain_history.Blocks, err error) {
	TryDo(func(attempt int) (retry bool, err error) {
		history, err = p.client.BlockchainHistory.GetBlocks(ctx, blockNum, limit)
		if err != nil {
			time.Sleep(p.Options.ErrorRetryTimeout)
		}
		return attempt < p.Options.ErrorRetryLimit, err
	})
	return
}
