package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"flag"

	"github.com/hashicorp/yamux"
)

func main() {

	if len(os.Args()) != 2 {
		log.Fatal("Please pass in host and port `echo <host> <port>`")
	}

	// establish a connection to the server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", os.Arg(0), os.Arg(1)))
	if err != nil {
		log.Fatal("Unable to establish connection to %s\nmake sure the relay server is reachable `relay --localForwardPort=<local_port> <host>:<port>\nError message: %s", err)
	}

	fmt.Fprintln(conn, "multiplex")

	// listen for the port we should report
	var serverPort string
	_, err = fmt.Fscanln(conn, &serverPort)
	if err != nil {
		log.Fatalf("Server response invalid %s", err)
	}

	fmt.Printf("established relay address: %s:%s\n", os.Arg(0), serverPort)

	// establish a multiplex server
	session, err := yamux.Server(conn, yamux.DefaultConfig())
	if err != nil {
		log.Fatalf("unable to multiplex the relay server connection %s", err)
	}

	// forward all connections to localForwardPort on localhost
	for {
		serverMuxConn, err := session.Accept()
		if err != nil {
			if err == io.EOF {
				log.Println("connection closed")
				return
			}
			log.Fatalf("Failed to accept connection from server %s", err)
		}

		go func(serverMuxConn net.Conn) {
			_, err = io.Copy(serverMuxConn, serverMuxConn)
			if err != nil && err != io.EOF {
				log.Printf("Echo Failure %s", err)
			}

		}(serverMuxConn)

	}
}
