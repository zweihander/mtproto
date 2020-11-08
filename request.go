package mtproto

type pendingRequest struct {
	response interface{}
	echan    chan error
}
