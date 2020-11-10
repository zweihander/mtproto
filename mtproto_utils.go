package mtproto

func (m *MTProto) AddCustomServerRequestHandler(handler func(i interface{}) bool) {
	m.serverRequestHandlers = append(m.serverRequestHandlers, handler)
}
