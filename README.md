# Packet Server

![Status: Work in Progress](https://img.shields.io/badge/Status-Work&#32;in&#32;Progress-blue.svg)

A simple, multi-threaded TCP/IP and UDP packet server framework written in Go. Originally forked
off of [firstrow/tcp_server](https://github.com/firstrow/tcp_server). Intended to be used as a
starting point for networking projects.

## Example Usage

``` go
package main

import "github.com/lukehollenback/tcp-server/tcp"

func main() {
  var err error

  //
  // Create a new server that will bind to port 9999.
  //
  server := tcp.CreateServer(&ServerConfig{
    address:             "localhost:9999",
    onNewClient: func(c *Client) { log.Print("Client connected.") },
    onNewMessage: func(c *Client, msg string) { log.Print(msg) },
    onClientConnectionClosed: func(client *Client) { log.Print("Client disconnected.") },
  })

  //
  // Bind the server and have it begin listening for connections.
  //
  err = server.Start()
  if err != nil {
    log.Fatalf("The server failed to start! (Error: %s)", err)
  }

  //
  // ...Do something here. Consider something like looping to handle operating system interupts...
  //

  //
  // Tell the server to stop and wait for it to finishe doing so.
  //
  var chStopped chan bool

  chStopped, err = server.Stop()
  if err != nil {
    log.Fatalf("The server failed to stop! (Error: %s)", err)
  }

  <-chStopped
}
```
