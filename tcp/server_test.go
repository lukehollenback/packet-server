package tcp

import (
	"net"
	"testing"
	"time"
)

const TestServerAddress = "localhost:9999"
const TestMessage = "This is a test message. Here is a number: 12345.67890!\n"

func TestBasicLifecycle(t *testing.T) {
	//
	// Define variables upon which we will state and assert proper functionality.
	//
	var messageReceived bool
	var messageText string
	var newClient bool
	var connectionClosed bool

	//
	// Create a new server and register event handlers that will set variables against which we can
	// run some assertions.
	//
	server, err := CreateServer(&ServerConfig{
		Address:     TestServerAddress,
		Delim:       '\x00',
		OnNewClient: func(c *Client) { newClient = true },
		OnNewMessage: func(c *Client, message string) {
			messageReceived = true
			messageText = message
		},
		OnClientConnectionClosed: func(client *Client) { connectionClosed = true },
	})
	if err != nil {
		t.Fatalf("The server failed to create. (Error: %s)", err)
	}

	server.Start()

	//
	// Give the server some time to bind.
	//
	time.Sleep(10 * time.Millisecond)

	//
	// Connect to the server as a new client and sent it a test message.
	//
	conn, err := net.Dial("tcp", TestServerAddress)

	if err != nil {
		t.Fatal("Failed to connect to the test server.")
	}

	conn.Write([]byte(TestMessage))

	conn.Close()

	//
	// Give the server a chance to recieve the new connection and the new message.
	//
	time.Sleep(10 * time.Millisecond)

	//
	// Assert that the server's handlers fired and that the expected messages came back.
	//
	if newClient != true {
		t.Error("The \"OnNewClient\" event handler never fired.")
	}

	if messageReceived != true {
		t.Error("The \"OnNewMessage\" event handler never fired.")
	}

	if messageText != TestMessage {
		t.Error("A message was recieved, but it was not equal to what was expected.")
	}

	if connectionClosed != true {
		t.Error("The \"OnClientConnectionClosed\" event handler never fired.")
	}

	//
	// Tell the server to shutdown and then wait for it to finish.
	//
	chStopped, _ := server.Stop()

	<-chStopped
}

func TestBasicLifecycleAgain(t *testing.T) {
	//
	// Define variables upon which we will state and assert proper functionality.
	//
	var messageReceived bool
	var messageText string
	var newClient bool
	var connectionClosed bool

	//
	// Create a new server and register event handlers that will set variables against which we can
	// run some assertions.
	//
	server, err := CreateServer(&ServerConfig{
		Address:     TestServerAddress,
		Delim:       '\x00',
		OnNewClient: func(c *Client) { newClient = true },
		OnNewMessage: func(c *Client, message string) {
			messageReceived = true
			messageText = message
		},
		OnClientConnectionClosed: func(client *Client) { connectionClosed = true },
	})
	if err != nil {
		t.Fatalf("The server failed to create. (Error: %s)", err)
	}

	server.Start()

	//
	// Give the server some time to bind.
	//
	time.Sleep(10 * time.Millisecond)

	//
	// Connect to the server as a new client and sent it a test message.
	//
	conn, err := net.Dial("tcp", TestServerAddress)

	if err != nil {
		t.Fatal("Failed to connect to the test server.")
	}

	conn.Write([]byte(TestMessage))

	conn.Close()

	//
	// Give the server a chance to recieve the new connection and the new message.
	//
	time.Sleep(10 * time.Millisecond)

	//
	// Assert that the server's handlers fired and that the expected messages came back.
	//
	if newClient != true {
		t.Error("The \"OnNewClient\" event handler never fired.")
	}

	if messageReceived != true {
		t.Error("The \"OnNewMessage\" event handler never fired.")
	}

	if messageText != TestMessage {
		t.Error("A message was recieved, but it was not equal to what was expected.")
	}

	if connectionClosed != true {
		t.Error("The \"OnClientConnectionClosed\" event handler never fired.")
	}

	//
	// Tell the server to shutdown and then wait for it to finish.
	//
	chStopped, _ := server.Stop()

	<-chStopped
}

func TestServerShutdownBeforeClientDisconnect(t *testing.T) {
	//
	// Create a new server and register event handlers that will set variables against which we can
	// run some assertions.
	//
	server, err := CreateServer(&ServerConfig{
		Address: TestServerAddress,
		Delim:   '\x00',
	})
	if err != nil {
		t.Fatalf("The server failed to create. (Error: %s)", err)
	}

	server.Start()

	//
	// Give the server some time to bind.
	//
	time.Sleep(10 * time.Millisecond)

	//
	// Connect to the server as a new client and sent it a test message.
	//
	conn, err := net.Dial("tcp", TestServerAddress)

	if err != nil {
		t.Fatal("Failed to connect to the test server.")
	}

	conn.Write([]byte(TestMessage))

	// NOTE: We explicitly do not close the client connection here.

	//
	// Give the server a chance to recieve the new connection and the new message.
	//
	time.Sleep(10 * time.Millisecond)

	//
	// Tell the server to shutdown and then wait for it to finish.
	//
	chStopped, _ := server.Stop()

	<-chStopped
}
