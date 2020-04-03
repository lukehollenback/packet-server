package tcpserver

import (
	"net"
	"testing"
	"time"
)

func buildTestServer() *server {
  return New("localhost:9999")
}

func TestOnNewClientCallback(t *testing.T) {
  var messageReceived bool
  var messageText string
  var newClient bool
  var connectinClosed bool

  //
  // Create a new server and register event handlers that will set variables against which we can
  // run some assertions.
  //
  server := buildTestServer()

  server.OnNewClient(func(c *Client) {
    newClient = true
  })
  server.OnNewMessage(func(c *Client, message string) {
    messageReceived = true
    messageText = message
  })
  server.OnClientConnectionClosed(func(c *Client, err error) {
    connectinClosed = true
  })
  go server.Listen()

  // Wait for server
  // If test fails - increase this value
  time.Sleep(10 * time.Millisecond)

  conn, err := net.Dial("tcp", "localhost:9999")
  if err != nil {
    t.Fatal("Failed to connect to test server")
  }
  conn.Write([]byte("Test message\n"))
  conn.Close()

  // Wait for server
  time.Sleep(10 * time.Millisecond)

  //
  // Assert that the handlers fired and that the expected messages came back.
  //
  if newClient != true {
    t.Error("The \"OnNewClient\" event handler never fired.")
  }

  if messageReceived != true {
    t.Error("The \"OnNewMessage\" event handler never fired.")
  }

  if messageText != "Test message\n" {
    t.Error("A message was recieved, but it was not equal to what was expected.")
  }

  if connectinClosed != true {
    t.Error("The \"OnClientConnectionClosed\" event handler never fired.")
  }
}
