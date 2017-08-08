package main

import (
	"fmt"
	"io"
	"flag"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

func localProxy() {
	if len(flag.Args()) != 2 {
		log.Fatalln("When using localForwardPort please specify server and port `relay --localForwardPort=<local_port> <host> <port>`")
	}

	// establish a connection to the server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", flag.Arg(0), flag.Arg(1)))
	if err != nil {
		log.Fatalf("Unable to establish connection to %s\nmake sure the relay server is reachable `relay --localForwardPort=<local_port> <host> <port>\nError message: %s", err)
	}

	// tell the server I want to use the multiplexer
	fmt.Fprintln(conn, "multiplex")

	// listen for the port we should report
	var serverPort string
	_, err = fmt.Fscanln(conn, &serverPort)
	if err != nil {
		log.Fatalf("Server response invalid %s", err)
	}

	fmt.Printf("established relay address: %s:%s", flag.Arg(0), serverPort)

	// establish a multiplex server
	session, err := yamux.Client(conn, yamux.DefaultConfig())
	if err != nil {
		log.Fatalf("unable to multiplex the relay server connection %s", err)
	}

	// forward all connections to localForwardPort on localhost
	for {
		serverMuxCon, err := session.Accept()
		if err != nil {
			log.Fatalf("Failed to accept connection from server %s", err)
			break
		}

		go func(serverMuxCon net.Conn) {

			defer serverMuxCon.Close()

			// establish a connection to the local forward port
			localForwardConn, err := net.Dial("tcp", fmt.Sprintf("localhost:%s", *localForwardPort))
			if err != nil {
				log.Fatalf("Failed to connect to local server on port %s %s", *localForwardPort, err)
			}

			go io.Copy(localForwardConn, serverMuxCon)
			_, err = io.Copy(serverMuxCon, localForwardConn)
			if err != nil && err != io.EOF {
				log.Printf("Error copying data between relay server and forward server %s", err)
			}
		}(serverMuxCon)

	}
}
