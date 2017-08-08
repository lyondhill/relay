package main

import (
	"net"
	"log"
	"io"
	"fmt"
	"crypto/rand"
	// "time"
)

// a set of all the connection waitingConnections connected to this node
var waitingConnections = map[string]net.Conn{}

func handleEventRequest(relayConn net.Conn) {

	// get the action requested by the client
	var action string
	_, err := fmt.Fscanln(relayConn, &action)
	if err != nil {
		log.Printf("Server response invalid %s", err)
		return
	}

	switch action {
	case "new":
		// establish a new listener
		listener, err := establishListener(relayConn)
		if err != nil {
			log.Fatal("Connection couldnt be established", err.Error())
		}

		// inform the connection of the newly established port number	
		_, err = fmt.Fprintf(relayConn, "%d\n", port(listener))
		if err != nil {
			log.Fatalf("Unable to communicate with the relay client %s", err)
		}

		go eventUserConnection(listener, relayConn)

	default:
		// close this new relay connection
		defer relayConn.Close()

		// or are they adding to a existing connection pool
		waitingConn, ok := waitingConnections[action]
		if !ok {
			// this this device was connectiong for a connection that is no longer waiting
			return
		}

		// we found the connection so lets remove it from the waiting list
		delete(waitingConnections, action)
		defer waitingConn.Close()

		// pipe data between relay and waiting connection
		go io.Copy(relayConn, waitingConn)
		io.Copy(waitingConn, relayConn)

	}	

}

// create a random id for the pool and confirm the pool hasnt already used this id
func randEventID() string {
    n := 5
    b := make([]byte, n)
    if _, err := rand.Read(b); err != nil {
        panic(err)
    }

    s := fmt.Sprintf("%X", b)
	
	// if the id is already used. try again
	if _, ok := waitingConnections[s]; ok {
		return randEventID()
	}
	return s
}

// handle all new incoming connection for this pool
func eventUserConnection(listener net.Listener, relayConn net.Conn) {
	// fire up the listener go routine and start reading from channels
	connChan := connChanListener(listener)

	// create a time so we can check to make sure the relay client is still connected
	// timeChan := time.Tick(10 * time.Second)

	// listen for new connections on the new listener
	for {
		select {
		case clientConn, ok := <-connChan:
			if !ok {
				// connections have stopped.
				return
			}
			go handleEventClientConn(clientConn, relayConn)

		}

	}

}

func handleEventClientConn(clientConn net.Conn, relayConn net.Conn) {
	id := randEventID()
	waitingConnections[id] = clientConn

	// let the relay know we have a connection waiting
	fmt.Fprintln(relayConn, id)
}
