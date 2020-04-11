package tcpserver

import (
	"bufio"
	"io"
	"log"
	"net"
)

//
// Client holds info about a single client connection.
//
type Client struct {
	conn   net.Conn // Literal connection to the client.
	server *Server  // The server that the client belongs to.
}

//
// RemoteAddr returns an address string (e.g. "{ip}:{port}") for the remote address of the client.
//
func (c *Client) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

//
// LocalAddr returns an address string (e.g. "{ip}:{port}") for the local address of the client.
//
func (c *Client) LocalAddr() string {
	return c.conn.LocalAddr().String()
}

//
// Close closes the current connection to the client.
//
func (c *Client) Close() {
	c.conn.Close()
}

//
// Send sends the specified message to the client.
//
func (c *Client) Send(message string) error {
	return c.SendBytes([]byte(message))
}

//
// SendBytes sends the specified bytes to the client.
//
func (c *Client) SendBytes(b []byte) error {
	_, err := c.conn.Write(b)
	return err
}

//
// listen reads and processes new messages from the client while it is connected. It is intended to
// be run in its own goroutine per connected client.
//
func (c *Client) listen() {
	//
	// Make sure that, even if something goes wrong, we close the client connection.
	//
	defer c.Close()

	//
	// Execute the registered "new client" event handler.
	//
	c.server.onNewClientCallback(c)

	//
	// Create a buffer reader to read recieved messages from the client and begin doing so in a loop.
	//
	reader := bufio.NewReader(c.conn)
	for {
		//
		// Attempt to block and read the next message from the client. If this fails for any reason
		// (e.g. an actual error or a disconnect), handle it accordingly.
		//
		message, err := reader.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				log.Printf("Client at %s has disconnected.", c.conn.RemoteAddr())
			} else {
				log.Printf("Buffer read for client at %s failed (%s). Connection will be closed.",
					c.conn.RemoteAddr(), err)
			}

			c.Close()
			c.server.onClientConnectionClosed(c, err)
			c.server.forgetClient(c)

			return
		}

		//
		// If we get this far, we recieved a valid message from the client. Thus, execute the registered
		// message handler.
		//
		c.server.onNewMessage(c, message)
	}
}