package tcp

import (
	"crypto/tls"
	"log"
	"net"
	"sync"
	"time"
)

//
// ServerConfig holds various configuration attributes for creating a new server.
//
type ServerConfig struct {
	Address                  string                           // The bind "{address}:{port}" for the server's listener.
	OnNewClient              func(client *Client)             // Handler function to execute when a new client connects.
	OnClientConnectionClosed func(client *Client)             // Handler function to execute when a client disconnects. Do not expect connection to still be alive when executed.
	OnNewMessage             func(client *Client, msg string) // Handler function to execute when a new message is recieved from a client.
}

//
// Server holds info about an actual server instance.
//
type Server struct {
	mu           *sync.Mutex     // Synchronizes access to the client table.
	config       *ServerConfig   // Basic configuration attributes of the server.
	tlsConfig    *tls.Config     // Secure connection configuration attributes of the server. Only relevent when using TLS.
	listener     net.Listener    // Actual listener that will bind to the configured address and await new connections.
	clients      map[int]*Client // Holds each connected client.
	nextClientID int             // Next valid client identifier that can be assigned to a new client.
	chStarted    chan bool       // Channel that will be used to tell whoever cares that the server has completed startup.
	chKill       chan bool       // Channel that will be used to tell the server's listener loop to stop.
	chStopped    chan bool       // Channel that will be used to tell whoever cares that the server's listener loop has stopped.
}

//
// SendAll sends the specified message to all clients currently connected to the server.
//
// NOTE: No synchronization is performed during this call. There is a chance that a send will be
//  attempted to a client that disconnects while this call executes.
//
func (o *Server) SendAll(msg string) {
	o.SendBytesAll([]byte(msg))
}

//
// SendBytesAll sends the specified bytes to all clients currently connected to the server.
//
// NOTE: No synchronization is performed during this call. There is a chance that a send will be
//  attempted to a client that disconnects while this call executes.
//
func (o *Server) SendBytesAll(pyld []byte) {
	// TODO: If enough clients are connected that it would matter, spin off a couple of goroutines and
	//  allocate them each a handful of the clients to send to.

	for _, client := range o.clients {
		err := client.SendBytes(pyld)

		if err != nil {
			log.Printf(
				"Failed to send \"send all\" message to a connected TCP/IP client. (Client: %s) (Hint: "+
					"The client may have already disconnected.)",
				client,
			)
		}
	}
}

//
// OnNewClient executes the server's registered "on new client" handler function.
//
func (o *Server) onNewClient(client *Client) {
	if o.config.OnNewClient == nil {
		return
	}

	o.config.OnNewClient(client)
}

//
// OnClientConnectionClosed executes the server's registered "on client connection closed" handler
// function.
//
func (o *Server) onClientConnectionClosed(client *Client) {
	if o.config.OnClientConnectionClosed == nil {
		return
	}

	o.config.OnClientConnectionClosed(client)
}

//
// OnNewMessage executes the server's registered "on new message" handler function.
//
func (o *Server) onNewMessage(client *Client, msg string) {
	if o.config.OnNewMessage == nil {
		return
	}

	o.config.OnNewMessage(client, msg)
}

//
// Start implements the method described by packetsvr.Server interface.
//
func (o *Server) Start() (<-chan bool, error) {
	//
	// Log some debug info.
	//
	log.Print("Attempting to start the TCP/IP packet server...")

	//
	// (Re)-initialize necessary members of the server structure.
	//
	o.clients = make(map[int]*Client, 0)
	o.chStarted = make(chan bool, 1)
	o.chKill = make(chan bool, 1)
	o.chStopped = make(chan bool, 1)

	//
	// Resolve the address.
	//
	tcpAddr, tcpAddrErr := net.ResolveTCPAddr("tcp", o.config.Address)
	if tcpAddrErr != nil {
		return nil, tcpAddrErr
	}

	//
	// Attempt to bind to the configured ip address and port.
	//
	var listenerErr error

	if o.tlsConfig == nil {
		o.listener, listenerErr = net.Listen("tcp", tcpAddr.String())
	} else {
		o.listener, listenerErr = tls.Listen("tcp", tcpAddr.String(), o.tlsConfig)
	}

	if listenerErr != nil {
		return nil, listenerErr
	}

	//
	// Fire up a goroutine to loop infinitely to accept new connections and spin off a handler thread
	// for each until the kill signal is sent.
	//
	go o.listen()

	///
	// Return a channel that can be blocked on if it is necessary to wait for the server to completely
	// start up.
	//
	// NOTE: The goroutine handling the server's lifecycle will send a message on the "started"
	//  channel that we return here once it has completely started up.
	//
	return o.chStarted, nil
}

//
// Stop implements the method described by packetsvr.Server interface.
//
func (o *Server) Stop() (<-chan bool, error) {
	//
	// Log some debug info.
	//
	log.Print("Attempting to stop the TCP/IP packet server...")

	//
	// Send the kill signal.
	//
	o.chKill <- true

	//
	// Return a channel that can be blocked on if it is necessary to wait for the server to completely
	// shutdown.
	//
	// NOTE: The goroutine handling the server's lifecycle will send a message on the "stopped"
	//  channel that we return here once it has completely shut down.
	//
	return o.chStopped, nil
}

//
// CreateServer creates a new regular server instance.
//
func CreateServer(config *ServerConfig) *Server {
	log.Print("Creating a TCP/IP packet server with address ", config.Address, ".")

	server := &Server{
		mu:        &sync.Mutex{},
		config:    config,
		tlsConfig: nil,
	}

	return server
}

//
// CreateServerWithTLS creates a new TLS-enabled server instance that can handle secure connections.
//
func CreateServerWithTLS(config *ServerConfig, certFile string, keyFile string) *Server {
	log.Print("Creating TLS-enabled TCP/IP packet server with address ", config.Address, ".")

	cert, _ := tls.LoadX509KeyPair(certFile, keyFile)
	tlsConfig := tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server := &Server{
		mu:        &sync.Mutex{},
		config:    config,
		tlsConfig: &tlsConfig,
	}

	return server
}

//
// getAndIncrementNextClientID returns the next unique identifier that can be assigned to a new
// client.
//
func (o *Server) getAndIncrementNextClientID() int {
	// NOTE:  We must lock because we are going to generate a new client identifier that must be
	//  unique, even if multiple goroutines are trying to do so around the same time.

	o.mu.Lock()
	defer o.mu.Unlock()

	id := o.nextClientID
	o.nextClientID++

	return id
}

//
// addClient adds the specified client to the server's client table.
//
func (o *Server) addClient(client *Client, id int) {
	// NOTE:  We must lock because we are going to mutate the client table. Multiple goroutines may
	//  be trying to perform this action around the same time.

	o.mu.Lock()
	defer o.mu.Unlock()

	o.clients[id] = client
}

//
// forgetClient removes the specified client from the server's client table (if it exists). Note
// that it does NOT close the connection to the client.
//
func (o *Server) forgetClient(c *Client) {
	// NOTE:  We must lock because we are going to mutate the client table. Multiple goroutines may
	//  be trying to perform this action around the same time.

	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.clients, c.ID())
}

//
// handleNewClient creates a new client structure to represent the provided connection, appends it
// to the server's client table, and spins off a new goroutine to handle future interactions with
// it.
//
func (o *Server) handleNewClient(conn net.Conn) {
	id := o.getAndIncrementNextClientID()
	client := CreateClient(id, conn, o)

	o.addClient(client, id)

	go client.listen()

	log.Printf("%sA TCP/IP client has connected.", client.LogPrefix())
}

//
// listen handles the entire running lifecycle of the server once started.
//
func (o *Server) listen() {
	//
	// Spin off a goroutine to listen for new connections.
	//
	chListener := make(chan net.Conn)
	chListenerDone := make(chan bool, 1)

	go func() {
		//
		// Attempt to block and listen for new connections. If an error occurs and it is temporary,
		// delay for a second and then continue listening. Otherwise, if it is not temporary, break out
		// and allow for shutdown to take place. Otherwise, provide the new connection on the
		// appropriate channel so that it can be handled.
		//
		for {
			conn, err := o.listener.Accept()
			if err != nil {
				if realErr, ok := err.(net.Error); ok && realErr.Temporary() {
					log.Printf(
						"A temporary error occured while listening for new TCP/IP connections. Will continue "+
							"listening after a short delay. (Error: %s)",
						err,
					)

					time.Sleep(1 * time.Second)
				} else {
					log.Printf(
						"A critical failure occurred while listening for new TCP/IP connections. (Error: %s) "+
							"(Hint: Was the server shut down?)",
						err,
					)

					break
				}
			} else {
				chListener <- conn
			}
		}

		close(chListener)

		chListenerDone <- true
	}()

	//
	// Indicate that the server has started.
	//
	o.chStarted <- true

	log.Print("The TCP/IP packet server has been started.")

	//
	// Select on either new connections or a kill signal.
	//
	stop := false

	for !stop {
		select {
		case conn, ok := <-chListener:
			if !ok {
				stop = true
			} else {
				o.handleNewClient(conn)
			}

		case <-o.chKill:
			stop = true
		}
	}

	//
	// Close the listener and block until the listener goroutine completes.
	//
	log.Print("Closing the TCP/IP packet server listener...")

	o.listener.Close()

	<-chListenerDone

	//
	// Disconnect all clients and wait for them to finish cleaning themselves up.
	//
	log.Printf("Disconnecting all %d clients from the TCP/IP packet server...", len(o.clients))

	for _, e := range o.clients {
		<-e.Close()
	}

	//
	// Log some debug info.
	//
	log.Print("The TCP/IP packet server has been stopped.")

	//
	// Tell anyone waiting on us that we are done.
	//
	o.chStopped <- true

	return
}
