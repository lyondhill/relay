package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
)

func handleRelayRequest(relayConn net.Conn) {
	i, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("unable to parse port number %s", os.Args[1])
	}

	// reguardless of the outcome of this function
	// we always want to close the proxy request
	defer relayConn.Close()

	// establish a new listener
	for ; i <= 65535; i++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", i))
		if err != nil {
			// we were unable to establish a connection on the port
			// so we will try the next available port
			continue
		}

		// inform the connection of the newly established port number
		_, err = fmt.Fprintf(relayConn, "%d\n", i)
		if err != nil {
			log.Printf("Unable to communicate with the relay client %s", err)
			return
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
				go handleClientConn(clientConn, session)

			case <-timeChan:
				if session.IsClosed() {
					listener.Close()
					break
				}
			}
		}

	}

	// if we get here we went through all the avaliable ports on the
	// machine and were unable to find an available one
	log.Fatalln("Unable to establish a relay listener")
}

// make the listener process a go routine so it can be shut down if the proxy shuts down
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

//
func handleClientConn(clientConn net.Conn, session *yamux.Session) {

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
