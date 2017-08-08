package main

import (
	"net"
	"log"
	"fmt"
	"time"
	"io"

	"github.com/hashicorp/yamux"
)

func handleMuxRequest(relayConn net.Conn) {

	// reguardless of the outcome of this function
	// we always want to close the proxy request
	defer relayConn.Close()

	// establish a new listener
	listener, err := establishListener(relayConn)
	if err != nil {
		log.Fatal(err.Error())
	}

	// inform the connection of the newly established port number	
	_, err = fmt.Fprintf(relayConn, "%d\n", port(listener))
	if err != nil {
		log.Fatalf("Unable to communicate with the relay client %s", err)
	}

	// switch to a multiplexed connection
	// because the proxy request client is actually the server
	// I will take the roll of a client
	session, err := yamux.Client(relayConn, yamux.DefaultConfig())
	if err != nil {
		log.Printf("unable to multiplex the relay client connection %s", err)
		return
	}

	// fire up the listener go routine and start reading from channels
	connChan := connChanListener(listener)

	// create a time so we can check to make sure the relay client is still connected
	timeChan := time.Tick(1 * time.Second)

	// listen for new connections on the new listener
	for {

		select {
		case clientConn, ok := <-connChan:
			if !ok {
				return
			}
			go handleMuxClientConn(clientConn, session)

		case <-timeChan:
			if session.IsClosed() {
				listener.Close()
				break
			}
		}
	}

	// if we get here we went through all the avaliable ports on the
	// machine and were unable to find an available one
	log.Fatalln("Unable to establish a relay listener")	
}

// once a new connection comes in for our client
// lets get it taken care of
func handleMuxClientConn(clientConn net.Conn, session *yamux.Session) {

	// ensure we close the client connection
	defer clientConn.Close()

	// establish new multiplexed connection
	relayMuxConn, err := session.Open()
	if err != nil {
		log.Printf("Unable to open a new multiplexed connection %s", err)
		return
	}

	// ensure we close the mux connection
	defer relayMuxConn.Close()

	// copy data through
	go io.Copy(relayMuxConn, clientConn)
	_, err = io.Copy(clientConn, relayMuxConn)
	if err != nil && err != io.EOF {
		log.Printf("Error copying data between server and relay %s", err)
	}
}