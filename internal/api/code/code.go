package code

const (
	InternalError = 5000 + iota
	BadRequest
	RPCError
	AuthError
	UpgraderError
	BroadCastError
	TooManyRequests
)

const (
	Success = 0
)
