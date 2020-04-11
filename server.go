package tcpserver

import (
	"crypto/tls"
	"log"
	"net"
	"time"
)

//
// Server holds info about an actual server instance.
//
type Server struct {
	running                  bool
	address                  string
	config                   *tls.Config
	listener                 net.Listener
	clients                  []*Client
	onNewClientCallback      func(c *Client)
	onClientConnectionClosed func(c *Client, err error)
	onNewMessage             func(c *Client, message string)
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
	// Initialize the necessary struct members and bind the server to the configured address and port.
	//
	s.initialize()

	//
	// Make sure that the server will always get cleaned up, no matter what happens to end execution.
	//
	defer s.listener.Close()

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
		s.handleNewClient(conn)
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
		log.Fatal("An attempt was made to stop a server that was not already running.")
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

//
// New creates a new regular server instance.
//
func New(address string) *Server {
	log.Print("Creating server with address ", address, ".")

	server := &Server{
		address: address,
		config:  nil,
	}

	server.OnNewClient(func(c *Client) {})
	server.OnNewMessage(func(c *Client, message string) {})
	server.OnClientConnectionClosed(func(c *Client, err error) {})

	return server
}

//
// NewWithTLS creates a new TLS-enabled server instance that can handle secure connections.
//
func NewWithTLS(address string, certFile string, keyFile string) *Server {
	log.Print("Creating server with address ", address, ".")

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

//
// initialize actually resets the server's internal state and binds it to the configured address
// and port so that it can begin handling new client connections.
//
func (s *Server) initialize() {
	//
	// Make sure that the server is not already running.
	//
	if s.running {
		log.Fatal("An attempt was made to start a server that was already running.")
	}

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
	// Set the running sentinel
	//
	s.running = true
}

//
// handleNewClient creates a new client structure to represent the provided connection, appends it
// to the server's client table, and spins off a new goroutine to handle future interactions with
// it.
//
func (s *Server) handleNewClient(conn net.Conn) {
	client := &Client{
		conn:   conn,
		server: s,
	}

	s.clients = append(s.clients, client)

	go client.listen()
}

//
// forgetClient removes the specified client from the server's client table (if it exists). Note
// that it does NOT close the connection to the client.
//
func (s *Server) forgetClient(c *Client) {
	for i, e := range s.clients {
		if e == c {
			s.clients[len(s.clients)-1], s.clients[i] = s.clients[i], s.clients[len(s.clients)-1]
			s.clients = s.clients[:len(s.clients)-1]

			return
		}
	}

	log.Fatal("An unknown client was specified to be forgotten by the server.")
}