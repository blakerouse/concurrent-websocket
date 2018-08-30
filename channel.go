package websocket

import (
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"
	"net"
)

type Channel struct {
	handler *Handler // Handle managing this connection.
	conn    net.Conn // Websocket connection.

	readDesc *netpoll.Desc // Read descriptor for netpoll.
}

// Create a new channel for the connection.
func newChannel(conn net.Conn, handler *Handler) *Channel {
	return &Channel{
		handler:  handler,
		conn:     conn,
		readDesc: netpoll.Must(netpoll.HandleReadOnce(conn)),
	}
}

// Close the channel.
func (c *Channel) Close() {
	c.handler.poller.Stop(c.readDesc)
	c.conn.Close()
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

// Read and process the message from the connection.
func (c *Channel) read() {
	data, op, err := wsutil.ReadClientData(c.conn)
	if err != nil {
		c.Close()
		return
	}

	c.handler.callback(c, OpCode(op), data)
}
