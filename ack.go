package mtproto

import "sync"

type ackStore struct {
	mtx  sync.Mutex
	acks map[int64]struct{}
}

func newAckStore() *ackStore {
	return &ackStore{
		acks: make(map[int64]struct{}),
	}
}

func (s *ackStore) Put(msgID int64) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.acks[msgID] = struct{}{}
}

func (s *ackStore) GotAck(msgID int64) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	delete(s.acks, msgID)
}

func (s *ackStore) GotMultipleAcks(msgIDs []int64) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for _, id := range msgIDs {
		delete(s.acks, id)
	}
}
