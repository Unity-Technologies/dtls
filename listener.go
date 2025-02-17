package dtls

import (
	"net"

	"github.com/pion/dtls/v2/internal/udp"
	"github.com/pion/dtls/v2/pkg/protocol"
	"github.com/pion/dtls/v2/pkg/protocol/recordlayer"
)

// Listen creates a DTLS listener
func Listen(network string, laddr *net.UDPAddr, config *Config) (net.Listener, error) {
	parent, err := checkConfigAndListenUDP(network, laddr, config)
	if err != nil {
		return nil, err
	}

	return &listener{
		config: config,
		parent: parent,
	}, nil
}

// NewListener creates a DTLS listener which accepts connections from an inner Listener.
func NewListener(inner net.Listener, config *Config) (net.Listener, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return &listener{
		config: config,
		parent: inner,
	}, nil
}

// listener represents a DTLS listener
type listener struct {
	config *Config
	parent net.Listener
}

// Accept waits for and returns the next connection to the listener.
// You have to either close or read on all connection that are created.
// Connection handshake will timeout using ConnectContextMaker in the Config.
// If you want to specify the timeout duration, set ConnectContextMaker.
func (l *listener) Accept() (net.Conn, error) {
	c, err := l.parent.Accept()
	if err != nil {
		return nil, err
	}
	return Server(c, l.config)
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
// Already Accepted connections are not closed.
func (l *listener) Close() error {
	return l.parent.Close()
}

// Addr returns the listener's network address.
func (l *listener) Addr() net.Addr {
	return l.parent.Addr()
}

func checkConfigAndListenUDP(network string, laddr *net.UDPAddr, config *Config) (net.Listener, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	lc := udp.ListenConfig{
		AcceptFilter: func(packet []byte) bool {
			pkts, err := recordlayer.UnpackDatagram(packet)
			if err != nil || len(pkts) < 1 {
				return false
			}
			h := &recordlayer.Header{}
			if err := h.Unmarshal(pkts[0]); err != nil {
				return false
			}
			return h.ContentType == protocol.ContentTypeHandshake
		},
	}
	return lc.Listen(network, laddr)
}

// ListenNoHandshake creates a DTLS listener which does not automatically handshake on Accept.
func ListenNoHandshake(network string, laddr *net.UDPAddr, config *Config) (net.Listener, error) {
	parent, err := checkConfigAndListenUDP(network, laddr, config)
	if err != nil {
		return nil, err
	}

	return &noHandshakeListener{
		listener{
			config: config,
			parent: parent,
		},
	}, nil
}

// noHandshakeListener represents a DTLS listener that does not automatically
// handshake on Accept.
type noHandshakeListener struct {
	listener
}

// Accept waits for and returns the next connection to the listener.
// Connection handshake does not occur during accept and needs to be manually
// called by passing the returned net.Conn to the Server function.
func (n *noHandshakeListener) Accept() (net.Conn, error) {
	return n.parent.Accept()
}
