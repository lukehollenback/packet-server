package tcpserver

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"time"
)

//
// Client holds info about a single client connection.
//
type Client struct {
	conn   net.Conn
	server *Server
}

//
// Server holds info about an actual server instance.
//
type Server struct {
	running                  bool
	address                  string // Address to open connection: localhost:9999
	config                   *tls.Config
	listener                 net.Listener
	clients                  []*Client
	onNewClientCallback      func(c *Client)
	onClientConnectionClosed func(c *Client, err error)
	onNewMessage             func(c *Client, message string)
}

// Read client data from channel
func (c *Client) listen() {
	c.server.onNewClientCallback(c)
	reader := bufio.NewReader(c.conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Buffer read for client at %s failed (%s).", c.conn.RemoteAddr(), err)

			c.conn.Close()

			c.server.onClientConnectionClosed(c, err)

			for i, e := range c.server.clients {
				if e == c {
					c.server.clients[len(c.server.clients)-1], c.server.clients[i] = c.server.clients[i], c.server.clients[len(c.server.clients)-1]
					c.server.clients = c.server.clients[:len(c.server.clients)-1]
				}
			}

			return
		}
		c.server.onNewMessage(c, message)
	}
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
// SendAll sends the specified message to all clients currently connected to the server. If any
// individual send operation fails, future sends will be terminated and the relevant error will be
// returned.
//
func (s *Server) SendAll(message string) error {
	return s.SendBytesAll([]byte(message))
}

//
// SendBytes sends the specified bytes to the client.
//
func (c *Client) SendBytes(b []byte) error {
	_, err := c.conn.Write(b)
	return err
}

//
// SendBytesAll sends the specified bytes to all clients currently connected to the server. If any
// individual send operation fails, future sends will be terminated and the relevant error will be
// returned.
//
func (s *Server) SendBytesAll(b []byte) error {
	// TODO: If enough clients are connected that it would matter, spin off a couple of goroutines and
	//  allocate them each a handful of the clients to send to.

	for _, e := range s.clients {
		err := e.SendBytes(b)

		if err != nil {
			return err
		}
	}

	return nil
}

//
// OnNewClient registers a function to be called immediately after the server accepts a new
// connection to a client and spins up a unique goroutine to handle communication with it.
//
func (s *Server) OnNewClient(callback func(c *Client)) {
	s.onNewClientCallback = callback
}

//
// OnClientConnectionClosed registers a function to be called immediately after a connection to a
// client is closed for any reason.
//
func (s *Server) OnClientConnectionClosed(callback func(c *Client, err error)) {
	s.onClientConnectionClosed = callback
}

//
// OnNewMessage registers a function to be called when a connected client receives a new message.
//
func (s *Server) OnNewMessage(callback func(c *Client, message string)) {
	s.onNewMessage = callback
}

//
// Start starts the server if it has not already been started, or errors otherwise.
//
func (s *Server) Start() {
	//
	// (Re)-initialize necessary members of the server structure.
	//
	s.clients = make([]*Client, 0)

	//
	// Attempt to fire up a the server.
	//
	var err error

	if s.config == nil {
		s.listener, err = net.Listen("tcp", s.address)
	} else {
		s.listener, err = tls.Listen("tcp", s.address, s.config)
	}

	if err != nil {
		log.Fatal("An error occurred while attempting to start the server (", err, ").")
	}

	//
	// Make sure that the server will always get cleaned up, no matter what happens to end execution.
	//
	defer s.listener.Close()

	//
	// Set the running sentinel
	//
	s.running = true

	//
	// Loop infinitely to accept new connections and spin off a handler thread for each.
	//
	for s.running {
		//
		// Attempt to block and listen for new connections. If an error occurs and it is temporary,
		// delay for a second and then continue listening. If it is not temporary, either continue
		// shutting down the server (if shutdown has beed started already), or otherwise panic.
		//
		conn, err := s.listener.Accept()

		if err != nil {
			if realErr, ok := err.(net.Error); ok && realErr.Temporary() {
				time.Sleep(1 * time.Second)
			} else if !s.running {
				log.Print("The server has stopped listening for new connections.")
			} else {
				log.Fatal(err)
			}

			continue
		}

		//
		// If we get this far, we have accepted a connection from a valid client. Create a structure to
		// represent said client and spin off a new goroutine to handle communication with it.
		//
		client := &Client{
			conn:   conn,
			server: s,
		}

		s.clients = append(s.clients, client)

		go client.listen()
	}
}

//
// Stop shuts down the running server.
//
func (s *Server) Stop() {
	//
	// Log some debug info.
	//
	log.Print("Attempting to stop server...")

	//
	// Make sure we even can stop the server (a.k.a. make sure that the server is actually running).
	//
	if !s.running {
		log.Print("There is no running server to stop.")
		return
	}

	//
	// Set the running sentinel to false. This will cause the server's listening loop to stop.
	//
	s.running = false

	//
	// Close all client connections and wait for them to recieve their appropriate socket EOF messages
	// and subsequently remove themselves from the slice of known clients.
	//
	for _, e := range s.clients {
		e.Close()
	}

	for {
		if len(s.clients) == 0 {
			log.Print("All clients have been disconnected.")
			break
		}
	}

	//
	// Unbind the server from its port and cause it to stop listening for new client connections.
	//
	s.listener.Close()

	//
	// Log some debug info.
	//
	log.Print("The server has been stopped.")
}

// Creates new tcp server instance
func New(address string) *Server {
	log.Println("Creating server with address", address)
	server := &Server{
		address: address,
		config:  nil,
	}

	server.OnNewClient(func(c *Client) {})
	server.OnNewMessage(func(c *Client, message string) {})
	server.OnClientConnectionClosed(func(c *Client, err error) {})

	return server
}

func NewWithTLS(address string, certFile string, keyFile string) *Server {
	log.Println("Creating server with address", address)
	cert, _ := tls.LoadX509KeyPair(certFile, keyFile)
	config := tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server := &Server{
		address: address,
		config:  &config,
	}

	server.OnNewClient(func(c *Client) {})
	server.OnNewMessage(func(c *Client, message string) {})
	server.OnClientConnectionClosed(func(c *Client, err error) {})

	return server
}
