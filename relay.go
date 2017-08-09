package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

// handle all incoming relay connections
func handleRelayRequest(relayConn net.Conn) {

	// get from the client what protocol it is interested in doing
	var protocol string
	_, err := fmt.Fscanln(relayConn, &protocol)
	if err != nil {
		log.Fatalf("failed to retrieve protocol %s", err)
	}

	switch protocol {
	case "multiplex":
		handleMuxRequest(relayConn)
	case "pool":
		handlePoolRequest(relayConn)
	case "event":
		handleEventRequest(relayConn)
	default:
		// disconnect the client. I didnt understand the request type
		return
	}

}

// attempt to establish a listener
// on established connection inform relayconn how to connect to it
func establishListener(relayConn net.Conn) (net.Listener, error) {

	// make the port unber more loopable
	i, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("unable to parse port number %s", os.Args[1])
	}

	// start at the port number we were given and attempt incremental increases
	for ; i <= 65535; i++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", i))
		if err != nil {
			// we were unable to establish a connection on the port
			// so we will try the next available port
			continue
		}

		return listener, nil
	}

	return nil, fmt.Errorf("Unable to establish a relay listener")
}

// Make the accept process a go routine so it can be shut down if the proxy shuts down
func connChanListener(listener net.Listener) chan net.Conn {
	netChannel := make(chan net.Conn)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {

				// if the error indicates the listener was closed
				// we just close the connection channel and return
				if !strings.Contains(err.Error(), "closed network connection") {
					log.Printf("Listener unable to accept new connections %s", err)
				}

				close(netChannel)
				return
			}
			netChannel <- conn
		}
	}()
	return netChannel
}

func port(listener net.Listener) int {
	addrParts := strings.Split(listener.Addr().String(), ":")
	port, _ := strconv.Atoi(addrParts[len(addrParts)-1])
	return port
}
