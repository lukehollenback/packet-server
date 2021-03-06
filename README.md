# Packet Server

<img src="https://img.shields.io/badge/status-Complete-green.svg" alt="Status: Complete"/>

A simple, multi-threaded TCP/IP packet server framework written in Go. Originally forked
off of [firstrow/tcp_server](https://github.com/firstrow/tcp_server). Intended to be used as a
starting point for networking projects.

## Example Usage

``` go
package main

import (
  "log"

  "github.com/lukehollenback/packet-server/tcp"
)

func main() {
  var err error

  //
  // Create a new server that will bind to port 9999.
  //
  server, err := tcp.CreateServer(&tcp.ServerConfig{
    Address:                  "localhost:9999",
    OnNewClient:              func(c *tcp.Client) { log.Print("Client connected.") },
    OnNewMessage:             func(c *tcp.Client, msg string) { log.Print(msg) },
    OnClientConnectionClosed: func(client *tcp.Client) { log.Print("Client disconnected.") },
    Delim:                    '\n',
  })
  if err != nil {
    log.Fatalf("The server failed to create. (Error: %s)", err)
  }


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
  // Tell the server to stop and wait for it to finish doing so.
  //
  chStopped, err := server.Stop()
  if err != nil {
    log.Fatalf("The server failed to stop! (Error: %s)", err)
  }

  <-chStopped
}
```
