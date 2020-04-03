# Golang TCP/IP Server

A simple multi-threaded TCP/IP packet server written in Go. Originally forked off of
[firstrow/tcp_server](https://github.com/firstrow/tcp_server).

## Example

``` go
package main

import "github.com/lukehollenback/tcp-server"

func main() {
	//
	// Create a new server that will bind to port 9999.
	//
	server := tcp_server.New("localhost:9999")

	//
	// Register a "new client connection" event handler. Whenever a new client connects, a new thread
	// (via a goroutine) will be fired up and will subsequently be resonsible for listening for and
	// handling any messages coming from it.
	//
	server.OnNewClient(func(c *tcp_server.Client) {
		c.Send("Hello")
	})

	//
	// Register a "new message from client" event handler. This will execute within the respective
	// client's associated goroutine whenever a new message is recieved.
	//
	server.OnNewMessage(func(c *tcp_server.Client, message string) {
		// new message received
	})

	//
	// Register a "client disconnected" event handler.
	//
	server.OnClientConnectionClosed(func(c *tcp_server.Client, err error) {
		// connection with client lost
	})

	//
	// Bind the server and have it begin listening for connections.
	//
	server.Listen()
}
```
