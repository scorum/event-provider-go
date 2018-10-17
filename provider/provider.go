package provider

import (
	"context"
	"sort"
	"time"

	"github.com/scorum/event-provider-go/event"
	"github.com/scorum/scorum-go"
	"github.com/scorum/scorum-go/apis/blockchain_history"
	"github.com/scorum/scorum-go/apis/chain"
	"github.com/scorum/scorum-go/transport/http"
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

type Provider struct {
	client          *scorumgo.Client
	Options         *Options
	CurrentBlockNum uint32
}

func NewProvider(url string, setters ...Option) *Provider {
	args := &Options{
		SyncInterval:          time.Second,
		BlocksHistoryMaxLimit: 100,
		ErrorRetryTimeout:     10 * time.Second,
		ErrorRetryLimit:       3,
	}

	for _, setter := range setters {
		setter(args)
	}

	transport := http.NewTransport(url)
	return &Provider{
		client:  scorumgo.NewClient(transport),
		Options: args,
	}
}

func (p *Provider) Provide(ctx context.Context, from, irreversibleFrom uint32, eventTypes []event.Type) (chan event.Block, chan event.Block, chan error) {
	if irreversibleFrom > from {
		log.Warn("EventProvider: irreversibleFrom > from")
	}

	blocksCh := make(chan event.Block)
	irreversibleBlocksCh := make(chan event.Block)
	errCh := make(chan error)
	go func(blocksCh, irreversibleBlocksCh chan event.Block, errCh chan error) {
		// genesis block
		if from == 0 {
			accounts, err := p.getExistingAccounts()
			if err != nil {
				errCh <- err
				return
			}

			// genesis block
			genesis := event.Block{
				BlockID:   "",
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
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
				properties, err := p.getChainProperties()
				if err != nil {
					errCh <- err
					return
				}

				if from >= properties.HeadBlockNumber {
					time.Sleep(p.Options.ErrorRetryTimeout)
					continue
				}

				// GetBlockHistory has descending order
				limit := properties.HeadBlockNumber - irreversibleFrom
				if limit > p.Options.BlocksHistoryMaxLimit {
					limit = p.Options.BlocksHistoryMaxLimit
				}

				offset := from + limit

				history, err := p.getBlockHistory(offset, limit)
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
						BlockID:   block.BlockID,
						BlockNum:  num,
						Timestamp: timestamp,
					}
					for _, transaction := range block.Transactions {
						for _, operation := range transaction.Operations {
							ev := event.ToEvent(operation)
							for _, eventType := range eventTypes {
								if ev.Type() == eventType {
									eBlock.Events = append(eBlock.Events, ev)
									break
								}
							}
						}
					}

					if len(eBlock.Events) != 0 {
						if num <= properties.LastIrreversibleBlockNumber && num > irreversibleFrom {
							irreversibleBlocksCh <- eBlock
						}

						if num > from {
							blocksCh <- eBlock
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

func (p *Provider) getExistingAccounts() ([]string, error) {
	const lookupAccountsMaxLimit = 1000

	lowerBound := ""
	result := make([]string, 0, lookupAccountsMaxLimit)

	for {
		accounts, err := p.lookupAccounts(lowerBound, lookupAccountsMaxLimit)
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

func (p *Provider) lookupAccounts(lowerBoundName string, limit uint16) (names []string, err error) {
	TryDo(func(attempt int) (retry bool, err error) {
		names, err = p.client.Database.LookupAccounts(lowerBoundName, limit)
		if err != nil {
			time.Sleep(p.Options.ErrorRetryTimeout)
		}
		return attempt < p.Options.ErrorRetryLimit, err
	})
	return
}

func (p *Provider) getChainProperties() (prop *chain.ChainProperties, err error) {
	TryDo(func(attempt int) (retry bool, err error) {
		prop, err = p.client.Chain.GetChainProperties()
		if err != nil {
			time.Sleep(p.Options.ErrorRetryTimeout)
		}
		return attempt < p.Options.ErrorRetryLimit, err
	})
	return
}

func (p *Provider) getBlockHistory(blockNum, limit uint32) (history blockchain_history.BlockHistory, err error) {
	TryDo(func(attempt int) (retry bool, err error) {
		history, err = p.client.BlockchainHistory.GetBlocksHistory(blockNum, limit)
		if err != nil {
			time.Sleep(p.Options.ErrorRetryTimeout)
		}
		return attempt < p.Options.ErrorRetryLimit, err
	})
	return
}
