package event

import (
	"errors"
	"time"

	"github.com/scorum/scorum-go/types"
)

var errWrongEventType = errors.New("wrong event type")

type converter func(operation types.Operation) Event

var converters map[types.OpType]converter

func init() {
	converters = map[types.OpType]converter{
		types.AccountCreateOpType:               toAccountCreateEvent,
		types.AccountCreateByCommitteeOpType:    toAccountCreateEvent,
		types.AccountCreateWithDelegationOpType: toAccountCreateEvent,
		types.VoteOpType:                        toVoteEvent,
		types.CommentOpType:                     toCommentEvent,
		types.DeleteCommentOpType:               toDeleteCommentEvent,
		types.CreateGame:                        toCreateGameEvent,
		types.CancelGame:                        toCancelGameEvent,
		types.UpdateGameStartTime:               toUpdateGameStartTimeEvent,
		types.PostGameResults:                   toPostGameResultsEvent,
		types.PostBet:                           toPostBetEvent,
		types.CancelPendingBets:                 toCancelPendingBetEvent,
		types.BetsMatched:                       toBetsMatchedEvent,
		types.GameStatusChanged:                 toGameStatusChangedEvent,
		types.BetResolved:                       toBetResolvedEvent,
		types.BetCancelled:                      toBetCancelledEvent,
		types.TransferOpType:                    toTransferEvent,
		types.CreateNFT:                         toCreateNFTEvent,
		types.UpdateNFTMetadata:                 toUpdateNFTMetadataEvent,
		types.CreateGameRound:                   toCreateGameRoundEvent,
		types.UpdateGameRoundResult:             toUpdateGameRoundResultEvent,
	}
}

type Event interface {
	Type() Type
}

func ToEvent(op types.Operation) Event {
	if converter, exists := converters[op.Type()]; exists {
		return converter(op)
	}

	return UnknownEvent{}
}

type Block struct {
	BlockNum  uint32
	Timestamp time.Time
	Events    []Event
}

// AccountCreateEvent
type AccountCreateEvent struct {
	Account string
}

func (e AccountCreateEvent) Type() Type {
	return AccountCreateEventType
}

func toAccountCreateEvent(op types.Operation) Event {
	account := ""
	switch v := op.(type) {
	case *types.AccountCreateOperation:
		account = v.NewAccountName
	case *types.AccountCreateByCommitteeOperation:
		account = v.NewAccountName
	case *types.AccountCreateWithDelegationOperation:
		account = v.NewAccountName
	default:
		panic(errWrongEventType)
	}

	return &AccountCreateEvent{
		Account: account,
	}
}

// VoteEvent
type VoteEvent struct {
	Voter    string
	Author   string
	PermLink string
	Weight   int16
}

func (e VoteEvent) Type() Type {
	return VoteEventType
}

func toVoteEvent(op types.Operation) Event {
	v, ok := op.(*types.VoteOperation)
	if !ok {
		panic(errWrongEventType)
	}

	if v.Weight < 0 {
		return &FlagEvent{
			Voter:    v.Voter,
			Author:   v.Author,
			PermLink: v.Permlink,
			Weight:   v.Weight,
		}
	} else {
		return &VoteEvent{
			Voter:    v.Voter,
			Author:   v.Author,
			PermLink: v.Permlink,
			Weight:   v.Weight,
		}
	}
}

// FlagsEvent
type FlagEvent struct {
	Voter    string
	Author   string
	PermLink string
	Weight   int16
}

func (e FlagEvent) Type() Type {
	return FlagEventType
}

// CommentEvent
type CommentEvent struct {
	PermLink       string
	ParentAuthor   string
	ParentPermLink string
	Author         string
	Body           string
	JsonMetadata   string
	Title          string
}

func (e CommentEvent) Type() Type {
	return CommentEventType
}

// PostEvent
type PostEvent struct {
	PermLink       string
	ParentPermLink string
	Author         string
	Body           string
	JsonMetadata   string
	Title          string
}

func (e PostEvent) Type() Type {
	return PostEventType
}

func toCommentEvent(op types.Operation) Event {
	v, ok := op.(*types.CommentOperation)
	if !ok {
		panic(errWrongEventType)
	}

	if v.ParentAuthor == "" {
		return &PostEvent{
			PermLink:       v.Permlink,
			ParentPermLink: v.ParentPermlink,
			Author:         v.Author,
			Body:           v.Body,
			JsonMetadata:   v.JsonMetadata,
			Title:          v.Title,
		}
	} else {
		return &CommentEvent{
			PermLink:       v.Permlink,
			ParentAuthor:   v.ParentAuthor,
			ParentPermLink: v.ParentPermlink,
			Author:         v.Author,
			Body:           v.Body,
			JsonMetadata:   v.JsonMetadata,
			Title:          v.Title,
		}
	}
}

// DeleteComment
type DeleteCommentEvent struct {
	PermLink string
	Author   string
}

func (e DeleteCommentEvent) Type() Type {
	return DeleteCommentEventType
}

func toDeleteCommentEvent(op types.Operation) Event {
	v, ok := op.(*types.DeleteCommentOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return &DeleteCommentEvent{
		PermLink: v.Permlink,
		Author:   v.Author,
	}
}

type CreateGameEvent struct {
	types.CreateGameOperation
}

func (e CreateGameEvent) Type() Type {
	return CreateGameEventType
}

func toCreateGameEvent(op types.Operation) Event {
	e, ok := op.(*types.CreateGameOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return CreateGameEvent{*e}
}

type CancelGameEvent struct {
	types.CancelGameOperation
}

func (e CancelGameEvent) Type() Type {
	return CancelGameEventType
}

func toCancelGameEvent(op types.Operation) Event {
	e, ok := op.(*types.CancelGameOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return CancelGameEvent{*e}
}

type UpdateGameStartTimeEvent struct {
	types.UpdateGameStartTimeOperation
}

func (e UpdateGameStartTimeEvent) Type() Type {
	return UpdateGameStartEventType
}

func toUpdateGameStartTimeEvent(op types.Operation) Event {
	e, ok := op.(*types.UpdateGameStartTimeOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return UpdateGameStartTimeEvent{*e}
}

type PostGameResultsEvent struct {
	types.PostGameResultsOperation
}

func (e PostGameResultsEvent) Type() Type {
	return PostGameResultsEventType
}

func toPostGameResultsEvent(op types.Operation) Event {
	e, ok := op.(*types.PostGameResultsOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return PostGameResultsEvent{*e}
}

type PostBetEvent struct {
	types.PostBetOperation
}

func (e PostBetEvent) Type() Type {
	return PostBetEventType
}

func toPostBetEvent(op types.Operation) Event {
	e, ok := op.(*types.PostBetOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return PostBetEvent{*e}
}

type CancelPendingBetEvent struct {
	types.CancelPendingBetsOperation
}

func (e CancelPendingBetEvent) Type() Type {
	return CancelPendingBetsEventType
}

func toCancelPendingBetEvent(op types.Operation) Event {
	e, ok := op.(*types.CancelPendingBetsOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return CancelPendingBetEvent{*e}
}

type BetsMatchedEvent struct {
	types.BetsMatchedVirtualOperation
}

func (e BetsMatchedEvent) Type() Type {
	return BetsMatchedEventType
}

func toBetsMatchedEvent(op types.Operation) Event {
	e, ok := op.(*types.BetsMatchedVirtualOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return BetsMatchedEvent{*e}
}

type GameStatusChangedEvent struct {
	types.GameStatusChangedVirtualOperation
}

func (e GameStatusChangedEvent) Type() Type {
	return GameStatusChangedEventType
}

func toGameStatusChangedEvent(op types.Operation) Event {
	e, ok := op.(*types.GameStatusChangedVirtualOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return GameStatusChangedEvent{*e}
}

type BetResolvedEvent struct {
	types.BetResolvedOperation
}

func (e BetResolvedEvent) Type() Type {
	return BetResolvedEventType
}

func toBetResolvedEvent(op types.Operation) Event {
	e, ok := op.(*types.BetResolvedOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return BetResolvedEvent{*e}
}

type BetCancelledEvent struct {
	types.BetCancelledOperation
}

func (e BetCancelledEvent) Type() Type {
	return BetCancelledEventType
}

func toBetCancelledEvent(op types.Operation) Event {
	e, ok := op.(*types.BetCancelledOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return BetCancelledEvent{*e}
}

type TransferEvent struct {
	types.TransferOperation
}

func (e TransferEvent) Type() Type {
	return TransferEventType
}

func toTransferEvent(op types.Operation) Event {
	e, ok := op.(*types.TransferOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return TransferEvent{*e}
}

type CreateNFTEvent struct {
	types.CreateNFTOperation
}

func (e CreateNFTEvent) Type() Type {
	return CreateGameEventType
}

func toCreateNFTEvent(op types.Operation) Event {
	e, ok := op.(*types.CreateNFTOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return CreateNFTEvent{*e}
}

type UpdateNFTMetadataEvent struct {
	types.UpdateNFTMetadataOperation
}

func (e UpdateNFTMetadataEvent) Type() Type {
	return UpdateNFTMetadataEventType
}

func toUpdateNFTMetadataEvent(op types.Operation) Event {
	e, ok := op.(*types.UpdateNFTMetadataOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return UpdateNFTMetadataEvent{*e}
}

type CreateGameRoundEvent struct {
	types.CreateGameRoundOperation
}

func (e CreateGameRoundEvent) Type() Type {
	return UpdateNFTMetadataEventType
}

func toCreateGameRoundEvent(op types.Operation) Event {
	e, ok := op.(*types.CreateGameRoundOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return CreateGameRoundEvent{*e}
}

type UpdateGameRoundResultEvent struct {
	types.UpdateGameRoundResultOperation
}

func (e UpdateGameRoundResultEvent) Type() Type {
	return UpdateNFTMetadataEventType
}

func toUpdateGameRoundResultEvent(op types.Operation) Event {
	e, ok := op.(*types.UpdateGameRoundResultOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return UpdateGameRoundResultEvent{*e}
}

type UnknownEvent struct{}

func (e UnknownEvent) Type() Type {
	return UnknownEventType
}
