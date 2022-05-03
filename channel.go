package websocket

import (
	"crypto/tls"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"
)

// Channel is a websocket connection that messages are read from and written to.
type Channel struct {
	handler *Handler // Handle managing this connection.
	conn    net.Conn // Websocket connection.

	readDesc *netpoll.Desc // Read descriptor for netpoll.

	// onClose is called when the channel is closed.
	onClose    func()
	onCloseMux sync.Mutex
}

// Create a new channel for the connection.
func newChannel(conn net.Conn, handler *Handler) *Channel {
	fdConn := conn
	tlsConn, ok := conn.(*tls.Conn)
	if ok {
		fdConn = tlsConn.NetConn()
	}
	return &Channel{
		handler:  handler,
		conn:     conn,
		readDesc: netpoll.Must(netpoll.HandleReadOnce(fdConn)),
	}
}

// Close the channel.
func (c *Channel) Close() {
	c.handler.poller.Stop(c.readDesc)
	c.conn.Close()
	c.onCloseMux.Lock()
	defer c.onCloseMux.Unlock()
	if c.onClose != nil {
		c.onClose()
	}
}

// Send a message over the channel. Once write concurrency of the handler is
// reached this method will block.
func (c *Channel) Send(op OpCode, data []byte) {
	c.handler.writePool.schedule(func() {
		err := wsutil.WriteServerMessage(c.conn, ws.OpCode(op), data)
		if err != nil {
			c.Close()
		}
	})
}

// SetOnClose sets the callback to get called when the channel is closed.
func (c *Channel) SetOnClose(callback func()) {
	c.onCloseMux.Lock()
	defer c.onCloseMux.Unlock()
	c.onClose = callback
}

// Read and process the message from the connection.
func (c *Channel) read() {
	data, op, err := wsutil.ReadClientData(c.conn)
	if err != nil {
		c.Close()
		return
	}

	c.handler.callback(c, OpCode(op), data)
}
