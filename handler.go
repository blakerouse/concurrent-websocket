package websocket

import (
	"net/http"

	"github.com/gobwas/ws"
	"github.com/mailru/easygo/netpoll"
)

// Handler is a high performance websocket handler that uses netpoll with
// read and write concurrency to allow a high number of concurrent websocket
// connections.
type Handler struct {
	callback  RecievedCallback
	poller    netpoll.Poller
	readPool  *pool
	writePool *pool
}

// OpCode represents an operation code.
type OpCode ws.OpCode

// Operation codes that will be used in the RecievedCallback. rfc6455 defines
// more operation codes but those are hidden by the implementation.
const (
	OpText   OpCode = 0x1
	OpBinary OpCode = 0x2
)

// RecievedCallback is the signature for the callback called when a message
// is recieved on a Channel.
type RecievedCallback func(c *Channel, op OpCode, data []byte)

// NewHandler creates a new websocket handler.
func NewHandler(callback RecievedCallback, readPoolConcurrency int, writePoolConcurrency int) (*Handler, error) {
	// Creates the poller that is used when a websocket connects. This is used
	// to prevent the spawning of a 2 goroutines per connection.
	poller, err := netpoll.New(nil)
	if err != nil {
		return nil, err
	}

	// Create the handle object.
	return &Handler{
		callback:  callback,
		poller:    poller,
		readPool:  newPool(readPoolConcurrency),
		writePool: newPool(writePoolConcurrency),
	}, nil
}

// UpgradeHandler upgrades the incoming http request to become a websocket
// connection.
func (h *Handler) UpgradeHandler(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		// Ignore the error, the UpdateHTTP handled notifying the client.
		return
	}

	// Open the channel with the client.
	ch := newChannel(conn, h)
	h.startRead(ch)
}

// Start the reading of the connection from epoll.
func (h *Handler) startRead(c *Channel) {
	err := h.poller.Start(c.readDesc, func(ev netpoll.Event) {
		// Verify the connection is not closed.
		if ev&netpoll.EventReadHup != 0 {
			// Connection closed.
			c.Close()
			return
		}

		// Trigger the read. This will block once read concurrency is hit.
		h.readPool.schedule(func() {
			// Actually read from the channel.
			c.read()

			// Resume to get the next message.
			err := h.poller.Resume(c.readDesc)
			if err != nil {
				// Failed to resume reading, close the connection.
				c.Close()
			}
		})
	})
	if err == netpoll.ErrRegistered {
		// Already being handled.
	} else if err == netpoll.ErrClosed {
		// Already closed.
	} else if err != nil {
		// Failed to start reading.
		c.Close()
	}
}
