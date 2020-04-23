package tcp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
)

//
// Client holds info about a single client connection.
//
type Client struct {
	id     int       // The unique id assigned to the client.
	conn   net.Conn  // Literal connection to the client.
	server *Server   // The server that the client belongs to.
	delim  byte      // The byte that should act as a message delimiter.
	chStop chan bool // Channel that will be used to tell the client's handler loop to stop.
	chDone chan bool // Channel that will be used to tell whoever cares that the client's handler loop has stopped.
}

//
// CreateClient instantiates and returns a new client instance.
//
func CreateClient(id int, conn net.Conn, server *Server, delim byte) *Client {
	o := &Client{
		id:     id,
		conn:   conn,
		server: server,
		delim:  delim,
		chStop: make(chan bool, 1),
		chDone: make(chan bool, 1),
	}

	return o
}

//
// String returns a printable representation of the client.
//
func (o *Client) String() string {
	return fmt.Sprintf("%05d TCP %21s", o.ID(), o.RemoteAddr())
}

//
// ID returns the unique id that has been assigned to client.
//
func (o *Client) ID() int {
	return o.id
}

//
// LogPrefix generates a prefix string that can be used in log messages about the client.
//
func (o *Client) LogPrefix() string {
	return o.logPrefix("  ")
}

//
// RcvLogPrefix generates a prefix string that can be used in log messages about messages recieved
// from the client.
//
func (o *Client) RcvLogPrefix() string {
	return o.logPrefix("~>")
}

//
// SndLogPrefix generates a prefix string that can be used in log messages about messages sent to
// the client.
//
func (o *Client) SndLogPrefix() string {
	return o.logPrefix("<~")
}

//
// RemoteAddr returns an address string (e.g. "{ip}:{port}") for the remote address of the client.
//
func (o *Client) RemoteAddr() string {
	return o.conn.RemoteAddr().String()
}

//
// LocalAddr returns an address string (e.g. "{ip}:{port}") for the local address of the client.
//
func (o *Client) LocalAddr() string {
	return o.conn.LocalAddr().String()
}

//
// Close beigns the process of closing the current connection to the client. It returns a channel
// that can optionally be blocked on if the caller would like to know when the connection has been
// completely closed.
//
func (o *Client) Close() <-chan bool {
	o.chStop <- true

	return o.chDone
}

//
// Send sends the specified message to the client.
//
func (o *Client) Send(message string) error {
	return o.SendBytes([]byte(message))
}

//
// SendBytes appends the appropriate delimiter and then sends the specified bytes to the client.
//
func (o *Client) SendBytes(b []byte) error {
	b = append(b, o.delim)

	_, err := o.conn.Write(b)

	return err
}

//
// logPrefix actually generates the prefix strings returned by the varous "*LogPrefix()" methods
// that are provided with public visibility.
//
func (o *Client) logPrefix(symbol string) string {
	return fmt.Sprintf("<~> %s %s ", o.String(), symbol)
}

//
// listen reads and processes new messages from the client while it is connected. It is intended to
// be run in its own goroutine per connected client.
//
func (o *Client) listen() {
	//
	// Execute the registered "new client" event handler.
	//
	o.server.onNewClient(o)

	//
	// Create a buffer reader to read recieved messages from the client and begin doing so in a new
	// goroutine.
	//
	reader := bufio.NewReader(o.conn)
	chReader := make(chan string)
	chReaderDone := make(chan bool, 1)

	go func() {
		for {
			msg, err := reader.ReadString(o.delim)

			if err != nil {
				if err == io.EOF {
					log.Printf("%sThe TCP/IP client has disconnected.", o.LogPrefix())
				} else {
					log.Printf(
						"%sBuffer read for the TCP/IP client failed. (Error: %s) (Hint: Did the server "+
							"shut down with clients still connected?)",
						o.LogPrefix(),
						err,
					)
				}

				break
			}

			chReader <- msg
		}

		close(chReader)

		chReaderDone <- true
	}()

	//
	// Select on either new messages or a kill signal.
	//
	stop := false

	for !stop {
		select {
		case msg, ok := <-chReader:
			if !ok {
				stop = true
			} else {
				o.server.onNewMessage(o, msg)
			}

		case <-o.chStop:
			stop = true
		}
	}

	//
	// Shutdown the connection.
	//
	o.server.onClientConnectionClosed(o)
	o.server.forgetClient(o)
	o.conn.Close()

	//
	// Block until the reader goroutine completes.
	//
	<-chReaderDone

	//
	// Tell anyone waiting on us that we are done.
	//
	o.chDone <- true

	return
}
