package socketio

import (
	"crypto/rand"
	"io"
	"net/http"
	"sync"
)

const (
	SessionIDLength  = 16
	SessionIDCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

func NewSessionID() string {
	b := make([]byte, SessionIDLength)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}

	for i := 0; i < SessionIDLength; i++ {
		b[i] = SessionIDCharset[b[i]%uint8(len(SessionIDCharset))]
	}

	return string(b)
}

type Session struct {
	mutex      sync.Mutex
	server     *SocketIOServer
	SessionId  string
	nameSpaces map[string]*NameSpace
	transport  Transport
	isConnect  bool
}

func NewSession(server *SocketIOServer, sessionId string) *Session {
	return &Session{server: server, SessionId: sessionId, nameSpaces: make(map[string]*NameSpace)}
}

func (ss *Session) Of(name string) (nameSpace *NameSpace) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	if nameSpace = ss.nameSpaces[name]; nameSpace == nil {
		nameSpace = NewNameSpace(ss, name)
		ss.nameSpaces[name] = nameSpace
	}
	return
}

func (ss *Session) serve(transportId string, w http.ResponseWriter, r *http.Request) {
	if ss.transport == nil {
		ss.transport = ss.server.transports.Get(transportId).New(ss)
	}
	ss.transport.OnData(w, r)
}

func (ss *Session) onFrame(data []byte) {
	packet, err := decodePacket(data)
	if err != nil {
		return
	}
	ss.onPacket(packet)
}

func (ss *Session) onPacket(packet Packet) {
	switch p := packet.(type) {
	case *disconnectPacket:
	case *connectPacket:
		ss.Of(packet.EndPoint()).onConnect()
	case *messagePacket, *jsonPacket:
		ss.Of(packet.EndPoint()).onMessagePacket(p.(messageMix))
	case *eventPacket:
		ss.Of(packet.EndPoint()).onEventPacket(p)
	}
}

func (ss *Session) onOpen() {
	if !ss.isConnect {
		packet := new(connectPacket)
		ss.Of("").sendPacket(packet)
	}
	ss.isConnect = true
}

// shortcut for Of("").(x)
func (ss *Session) On(name string, fn interface{}) error {
	return ss.Of("").On(name, fn)
}

func (ss *Session) RemoveListener(name string, fn interface{}) {
	ss.Of("").RemoveListener(name, fn)
}

func (ss *Session) Once(name string, fn interface{}) error {
	return ss.Of("").Once(name, fn)
}

func (ss *Session) RemoveAllListeners(name string) {
	ss.Of("").RemoveAllListeners(name)
}

func (ss *Session) emit(name string, callback func([]interface{}), args ...interface{}) {
	ss.Of("").emit(name, callback, args...)
}

func (ss *Session) emitRaw(name string, callback func([]interface{}), data []byte) error {
	return ss.Of("").emitRaw(name, callback, data)
}
