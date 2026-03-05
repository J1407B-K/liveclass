package code

const (
	InternalError = 5000 + iota
	BadRequest
	RPCError
	AuthError
	UpgraderError
	BroadCastError
)

const (
	Success = 0
)
