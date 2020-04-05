package tcpserver

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
)

// Client holds info about connection
type Client struct {
	conn   net.Conn
	Server *server
}

// TCP server
type server struct {
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
	c.Server.onNewClientCallback(c)
	reader := bufio.NewReader(c.conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			c.conn.Close()
			c.Server.onClientConnectionClosed(c, err)
			return
		}
		c.Server.onNewMessage(c, message)
	}
}

func (c *Client) close() {
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
func (s *server) SendAll(message string) error {
	for _, e := range s.clients {
		err := e.Send(message)

		if err != nil {
			return err
		}
	}

	return nil
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
func (s *server) SendBytesAll(b []byte) error {
	for _, e := range s.clients {
		err := e.SendBytes(b)

		if err != nil {
			return err
		}
	}

	return nil
}

//
// Conn returns the current connection to the client.
//
func (c *Client) Conn() net.Conn {
	return c.conn
}

//
// Close closes the current connection to the client.
//
func (c *Client) Close() error {
	return c.conn.Close()
}

// Called right after server starts listening new client
func (s *server) OnNewClient(callback func(c *Client)) {
	s.onNewClientCallback = callback
}

// Called right after connection closed
func (s *server) OnClientConnectionClosed(callback func(c *Client, err error)) {
	s.onClientConnectionClosed = callback
}

// Called when Client receives new message
func (s *server) OnNewMessage(callback func(c *Client, message string)) {
	s.onNewMessage = callback
}

//
// Start starts the server if it has not already been started, or errors otherwise.
//
func (s *server) Start() {
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
		log.Fatal("Error starting TCP server.")
	}

	//
	// Make sure that the server will always get cleaned up, no matter what happens to end execution.
	//
	defer s.listener.Close()

	//
	// Loop infinitely to accept new connections and spin off a handler thread for each.
	//
	for {
		conn, _ := s.listener.Accept()

		client := &Client{
			conn:   conn,
			Server: s,
		}

		s.clients = append(s.clients, client)

		go client.listen()
	}
}

func (s *server) Stop() {
	for _, e := range s.clients {
		e.Close()
	}

	s.listener.Close()
}

// Creates new tcp server instance
func New(address string) *server {
	log.Println("Creating server with address", address)
	server := &server{
		address: address,
		config:  nil,
	}

	server.OnNewClient(func(c *Client) {})
	server.OnNewMessage(func(c *Client, message string) {})
	server.OnClientConnectionClosed(func(c *Client, err error) {})

	return server
}

func NewWithTLS(address string, certFile string, keyFile string) *server {
	log.Println("Creating server with address", address)
	cert, _ := tls.LoadX509KeyPair(certFile, keyFile)
	config := tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server := &server{
		address: address,
		config:  &config,
	}

	server.OnNewClient(func(c *Client) {})
	server.OnNewMessage(func(c *Client, message string) {})
	server.OnClientConnectionClosed(func(c *Client, err error) {})

	return server
}
