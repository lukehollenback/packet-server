package tcpserver

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
)

//
// Client holds info about a single client connection.
//
type Client struct {
	conn   net.Conn
	Server *Server
}

//
// Server holds info about an actual server instance.
//
type Server struct {
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

//
// Close closes the current connection to the client.
//
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
// Conn returns the current connection to the client.
//
func (c *Client) Conn() net.Conn {
	return c.conn
}

// Called right after server starts listening new client
func (s *Server) OnNewClient(callback func(c *Client)) {
	s.onNewClientCallback = callback
}

// Called right after connection closed
func (s *Server) OnClientConnectionClosed(callback func(c *Client, err error)) {
	s.onClientConnectionClosed = callback
}

// Called when Client receives new message
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

func (s *Server) Stop() {
	for _, e := range s.clients {
		e.Close()
	}

	s.listener.Close()
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
