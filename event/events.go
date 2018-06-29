package event

import (
	"errors"

	"github.com/scorum/scorum-go/types"
)

var errWrongEventType = errors.New("wrong type")

type converter func(operation types.Operation, blockID string, blockNum uint32) Event

var converters map[types.OpType]converter

func init() {
	converters = map[types.OpType]converter{
		types.AccountCreateOpType:               toAccountCreateEvent,
		types.AccountCreateByCommitteeOpType:    toAccountCreateEvent,
		types.AccountCreateWithDelegationOpType: toAccountCreateEvent,
		types.VoteOpType:                        toVoteEvent,
		types.CommentOpType:                     toCommentEvent,
	}
}

type Event interface {
	Type() Type
	Common() CommonEvent
}

func ToEvent(op types.Operation, blockID string, blockNum uint32) Event {
	if converter, exists := converters[op.Type()]; exists {
		return converter(op, blockID, blockNum)
	}

	return toCommonEvent(op, blockID, blockNum)
}

type CommonEvent struct {
	BlockID  string
	BlockNum uint32
}

func (e CommonEvent) Type() Type {
	return UnknownEventType
}

func (e CommonEvent) Common() CommonEvent {
	return e
}

func toCommonEvent(_ types.Operation, blockID string, blockNum uint32) Event {
	return &CommonEvent{
		BlockID:  blockID,
		BlockNum: blockNum,
	}
}

type AccountCreateEvent struct {
	CommonEvent
	Account string
}

func (e AccountCreateEvent) Type() Type {
	return AccountCreateEventType
}

func (e AccountCreateEvent) Common() CommonEvent {
	return e.CommonEvent
}

func toAccountCreateEvent(op types.Operation, blockID string, blockNum uint32) Event {
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
		CommonEvent: CommonEvent{
			BlockID:  blockID,
			BlockNum: blockNum,
		},
		Account: account,
	}
}

type VoteEvent struct {
	CommonEvent
	Voter    string
	Author   string
	PermLink string
	Weight   int16
}

func (e VoteEvent) Type() Type {
	return VoteEventType
}

type FlagEvent struct {
	CommonEvent
	Voter    string
	Author   string
	PermLink string
	Weight   int16
}

func (e FlagEvent) Type() Type {
	return FlagEventType
}

func (e FlagEvent) Common() CommonEvent {
	return e.CommonEvent
}

func toVoteEvent(op types.Operation, blockID string, blockNum uint32) Event {
	v, ok := op.(*types.VoteOperation)
	if !ok {
		panic(errWrongEventType)
	}

	if v.Weight < 0 {
		return &FlagEvent{
			CommonEvent: CommonEvent{
				BlockID:  blockID,
				BlockNum: blockNum,
			},
			Voter:    v.Voter,
			Author:   v.Author,
			PermLink: v.Permlink,
			Weight:   v.Weight,
		}
	} else {
		return &VoteEvent{
			CommonEvent: CommonEvent{
				BlockID: blockID,
			},
			Voter:    v.Voter,
			Author:   v.Author,
			PermLink: v.Permlink,
			Weight:   v.Weight,
		}
	}
}

type CommentEvent struct {
	CommonEvent
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

func (e CommentEvent) Common() CommonEvent {
	return e.CommonEvent
}

type PostEvent struct {
	CommonEvent
	ParentPermLink string
	Author         string
	Body           string
	JsonMetadata   string
	Title          string
}

func (e PostEvent) Type() Type {
	return PostEventType
}

func (e PostEvent) Common() CommonEvent {
	return e.CommonEvent
}

func toCommentEvent(op types.Operation, blockID string, blockNum uint32) Event {
	v, ok := op.(*types.CommentOperation)
	if !ok {
		panic(errWrongEventType)
	}

	if v.ParentAuthor == "" {
		return &PostEvent{
			CommonEvent: CommonEvent{
				BlockID:  blockID,
				BlockNum: blockNum,
			},
			ParentPermLink: v.ParentPermlink,
			Author:         v.Author,
			Body:           v.Body,
			JsonMetadata:   v.JsonMetadata,
			Title:          v.Title,
		}
	} else {
		return &CommentEvent{
			CommonEvent: CommonEvent{
				BlockID: blockID,
			},
			ParentAuthor:   v.ParentAuthor,
			ParentPermLink: v.ParentPermlink,
			Author:         v.Author,
			Body:           v.Body,
			JsonMetadata:   v.JsonMetadata,
			Title:          v.Title,
		}
	}
}
