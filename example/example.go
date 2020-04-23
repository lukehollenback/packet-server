package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/lukehollenback/packet-server/tcp"
)

func main() {
	var err error
	var ch <-chan bool

	//
	// Register a kill signal handler with the operating system so that we can gracefully shutdown if
	// necessary.
	//
	osInterrupt := make(chan os.Signal, 1)

	signal.Notify(osInterrupt, os.Interrupt)

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
	ch, err = server.Start()
	if err != nil {
		log.Fatalf("The server failed to start! (Error: %s)", err)
	}

	<-ch

	//
	// Block until we are shut down by the operating system.
	//
	<-osInterrupt

	//
	// Tell the server to stop and wait for it to finishe doing so.
	//
	ch, err = server.Stop()
	if err != nil {
		log.Fatalf("The server failed to stop! (Error: %s)", err)
	}

	<-ch
}
