package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

// get a local port to possibly forward connections to
// this can be used to allow users to use the relay as a relay client/proxy
// so servers can run without doing anything special to account for the new relay
var localForwardPort = flag.String("localForwardPort", "", "local port to forward connections to")

func main() {
	flag.Parse()
	// if we were given a local forward port we will
	// act as a client and communicate with the server
	// on behalf of a local server listening on localForwardPort
	if *localForwardPort != "" {
		localProxy()
		return
	}

	if len(flag.Args()) != 1 {
		log.Fatalln("Please specify a listen port `relay <port>`")
	}

	// if we are setup we are good to start the server
	serverStart()

}

// initiate the server
func serverStart() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", flag.Arg(0)))
	if err != nil {
		log.Fatalf("Cannot establish listener: %s\n", err)
	}

	// log.Printf("connection estableshed on %s", os.Args[1])
	// log.Println("waiting for incoming relay requests")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Unable to accept connection: %s\n", err)
			continue
		}

		// handle the new relay request
		// since all connections are relay requests
		go handleRelayRequest(conn)
	}

}
