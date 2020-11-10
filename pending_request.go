package mtproto

import (
	"fmt"
	"sync"
)

type pendingRequest struct {
	// в response хранится тип который ожидаем получить
	// если он nil, то эта структурка не попадет в мапу pendingRequests
	response interface{}
	echan    chan error
}

type requestStore struct {
	mtx      sync.Mutex
	requests map[int64]pendingRequest
}

func newRequestStore() *requestStore {
	return &requestStore{
		requests: make(map[int64]pendingRequest),
	}
}

func (rs *requestStore) Put(messageID int64, req pendingRequest) error {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	if _, found := rs.requests[messageID]; found {
		return fmt.Errorf("request with provided messageID already exists: %d", messageID)
	}

	rs.requests[messageID] = req
	return nil
}

func (rs *requestStore) Pop(messageID int64) (pendingRequest, bool) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	req, found := rs.requests[messageID]
	if found {
		delete(rs.requests, messageID)
		return req, true
	}

	return req, false
}

// Note:
// итератор не должен сохранять request-ы после своей работы
func (rs *requestStore) ForEach(iter func(msgID int64, req pendingRequest)) {
	rs.mtx.Lock()
	defer rs.mtx.Unlock()

	for id, req := range rs.requests {
		iter(id, req)
	}
}
