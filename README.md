# Golang TCP/IP Server

![Status: Work in Progress](https://img.shields.io/badge/Status-Work&dash;in&dash;Progress-blue.svg)

A simple multi-threaded TCP/IP packet server written in Go. Originally forked off of
[firstrow/tcp_server](https://github.com/firstrow/tcp_server). Intended to be used as a starting
point for networking projects.

## Example Usage

``` go
package main

import "github.com/lukehollenback/tcp-server"

func main() {
  //
  // Create a new server that will bind to port 9999.
  //
  server := tcpserver.New("localhost:9999")

  //
  // Register a "new client connection" event handler. Whenever a new client connects, a new thread
  // (via a goroutine) will be fired up and will subsequently be resonsible for listening for and
  // handling any messages coming from it.
  //
  server.OnNewClient(func(c *tcpserver.Client) {
    c.Send("Hello")
  })

  //
  // Register a "new message from client" event handler. This will execute within the respective
  // client's associated goroutine whenever a new message is recieved.
  //
  server.OnNewMessage(func(c *tcpserver.Client, message string) {
    // new message received
  })

  //
  // Register a "client disconnected" event handler.
  //
  server.OnClientConnectionClosed(func(c *tcpserver.Client, err error) {
    // connection with client lost
  })

  //
  // Bind the server and have it begin listening for connections.
  //
  server.Listen()
}
```
