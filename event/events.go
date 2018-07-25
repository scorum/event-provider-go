package event

import (
	"errors"

	"github.com/scorum/scorum-go/types"
	"time"
)

var errWrongEventType = errors.New("wrong event type")

type converter func(operation types.Operation, commonEvent CommonEvent) Event

var converters map[types.OpType]converter

func init() {
	converters = map[types.OpType]converter{
		types.AccountCreateOpType:               toAccountCreateEvent,
		types.AccountCreateByCommitteeOpType:    toAccountCreateEvent,
		types.AccountCreateWithDelegationOpType: toAccountCreateEvent,
		types.VoteOpType:                        toVoteEvent,
		types.CommentOpType:                     toCommentEvent,
		types.DeleteCommentOpType:               toDeleteCommentEvent,
	}
}

type Event interface {
	Type() Type
	Common() CommonEvent
}

func ToEvent(op types.Operation, blockID string, blockNum uint32, timestamp time.Time) Event {
	commonEvent := toCommonEvent(op, blockID, blockNum, timestamp)

	if converter, exists := converters[op.Type()]; exists {
		return converter(op, *commonEvent)
	}

	return commonEvent
}

// Common Event
type CommonEvent struct {
	BlockID   string
	BlockNum  uint32
	Timestamp time.Time
}

func (e CommonEvent) Type() Type {
	return UnknownEventType
}

func (e CommonEvent) Common() CommonEvent {
	return e
}

func toCommonEvent(_ types.Operation, blockID string, blockNum uint32, timestamp time.Time) *CommonEvent {
	return &CommonEvent{
		BlockID:   blockID,
		BlockNum:  blockNum,
		Timestamp: timestamp,
	}
}

// Account Create Event
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

func toAccountCreateEvent(op types.Operation, commonEvent CommonEvent) Event {
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
		CommonEvent: commonEvent,
		Account:     account,
	}
}

// Vote Event
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

func toVoteEvent(op types.Operation, commonEvent CommonEvent) Event {
	v, ok := op.(*types.VoteOperation)
	if !ok {
		panic(errWrongEventType)
	}

	if v.Weight < 0 {
		return &FlagEvent{
			CommonEvent: commonEvent,
			Voter:       v.Voter,
			Author:      v.Author,
			PermLink:    v.Permlink,
			Weight:      v.Weight,
		}
	} else {
		return &VoteEvent{
			CommonEvent: commonEvent,
			Voter:       v.Voter,
			Author:      v.Author,
			PermLink:    v.Permlink,
			Weight:      v.Weight,
		}
	}
}

// Flag Event
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

// Comment Event
type CommentEvent struct {
	CommonEvent
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

func (e CommentEvent) Common() CommonEvent {
	return e.CommonEvent
}

// Post Event
type PostEvent struct {
	CommonEvent
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

func (e PostEvent) Common() CommonEvent {
	return e.CommonEvent
}

func toCommentEvent(op types.Operation, commonEvent CommonEvent) Event {
	v, ok := op.(*types.CommentOperation)
	if !ok {
		panic(errWrongEventType)
	}

	if v.ParentAuthor == "" {
		return &PostEvent{
			CommonEvent:    commonEvent,
			PermLink:       v.Permlink,
			ParentPermLink: v.ParentPermlink,
			Author:         v.Author,
			Body:           v.Body,
			JsonMetadata:   v.JsonMetadata,
			Title:          v.Title,
		}
	} else {
		return &CommentEvent{
			CommonEvent:    commonEvent,
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

// Delete Comment
type DeleteCommentEvent struct {
	CommonEvent
	PermLink       string
	ParentPermLink string
	Author         string
}

func (e DeleteCommentEvent) Type() Type {
	return DeleteCommentEventType
}

func (e DeleteCommentEvent) Common() CommonEvent {
	return e.CommonEvent
}

func toDeleteCommentEvent(op types.Operation, commonEvent CommonEvent) Event {
	v, ok := op.(*types.DeleteCommentOperation)
	if !ok {
		panic(errWrongEventType)
	}

	return &DeleteCommentEvent{
		CommonEvent:    commonEvent,
		PermLink:       v.Permlink,
		Author:         v.Author,
	}
}
