package event

type Type int

const (
	UnknownEventType Type = iota
	AccountCreateEventType
	PostEventType
	CommentEventType
	VoteEventType
	FlagEventType
	DeleteCommentEventType
)
