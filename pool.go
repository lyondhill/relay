package main

import (
	"net"
	"log"
	"sync"
	"io"
	"fmt"
	"crypto/rand"
	"time"
)

// a set of all the connection pools connected to this node
var pools = map[string]sync.Pool{}

func handlePoolRequest(relayConn net.Conn) {

	// get the action requested by the client
	var action string
	_, err := fmt.Fscanln(relayConn, &action)
	if err != nil {
		log.Printf("Server response invalid %s", err)
		relayConn.Close()
		return
	}

	switch action {
	case "new":
		// is this connection for a new service
		connPool := sync.Pool{}
		connPool.Put(relayConn)


		// generate a new id for this pool
		id := randPoolID()
		pools[id] = connPool

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


		// inform the new connection of the ID they can use
		_, err = fmt.Fprintf(relayConn, "%s\n", id)
		if err != nil {
			log.Fatalf("Unable to communicate with the relay client %s", err)
		}

		go poolUserConnection(listener, id)

	default:
		// or are they adding to a existing connection pool
		pool, ok := pools[action]
		if !ok {
			// this pool was not created so we should reject the connection
			relayConn.Close()
			return
		}
		pool.Put(relayConn)
		pools[action] = pool
	}	

}

// create a random id for the pool and confirm the pool hasnt already used this id
func randPoolID() string {
    n := 5
    b := make([]byte, n)
    if _, err := rand.Read(b); err != nil {
        panic(err)
    }

    s := fmt.Sprintf("%X", b)
	
	if _, ok := pools[s]; ok {
		return randPoolID()
	}
	return s
}

// handle all new incoming connection for this pool
func poolUserConnection(listener net.Listener, id string) {

	// fire up the listener go routine and start reading from channels
	connChan := connChanListener(listener)

	// create a time so we can check to make sure the relay client is still connected
	timeChan := time.Tick(10 * time.Second)

	// listen for new connections on the new listener
	for {
		select {
		case clientConn, ok := <-connChan:
			if !ok {
				// connections have stopped.
				// clean the pool for this id
				delete(pools, id)
				return
			}
			go handlePoolClientConn(clientConn, id)

		case <-timeChan:
			fmt.Println("timer")
			// check to see if the connection pool is empty.
			// if it is. we can assume the relay is done
			pool := pools[id]
			connection := pool.Get()
			if connection == nil {
				// shut down the listener 
				listener.Close()
				return
			} else {
				// put the connection back in the pool
				pool.Put(connection)
			}
		}
	}
	
}

func handlePoolClientConn(clientConn net.Conn, id string) {
	// dont leave the client connection open
	defer clientConn.Close() 

	// grab a new connection from the pool.
	pool, ok := pools[id]
	if !ok {
		// if the pool doesnt exist any more.. neither shoudl this connection
		clientConn.Close()
		return
	}

	// poll the pool until we get a available connection
	relayConnInterface := pool.Get()
	for relayConnInterface == nil {
		<-time.After(1 * time.Second)
		relayConnInterface = pool.Get()
	}

	relayConn := relayConnInterface.(net.Conn)

	// copy data to and from the pool connection
	go io.Copy(clientConn, relayConn)
	io.Copy(relayConn, clientConn)

	// close the pool connection
	relayConn.Close()
}
