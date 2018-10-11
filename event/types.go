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
	CreateGameEventType
	CancelGameEventType
	UpdateGameStartEventType
	PostGameResultsEventType
	PostBetEventType
	CancelPendingBetsEventType
	BetsMatchedEventType
)
