package main

import (
	"fmt"
	"io"
	"flag"
	"net"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
)

func init() {
	os.Args = []string{"relay", "1234"}
	go serverStart()
}

func TestMuxServer(t *testing.T) {

	serv, _ := net.Dial("tcp", "localhost:1234")

	fmt.Fprintln(serv, "multiplex")

	var serverPort string
	fmt.Fscanln(serv, &serverPort)
	if serverPort != "1235" {
		t.Errorf("Server reported an inaccurate port: %s", serverPort)
	}

	session, _ := yamux.Server(serv, yamux.DefaultConfig())

	msgs := make(chan string)
	go func() {
		servConn, _ := session.Accept()

		var info string
		fmt.Fscanln(servConn, &info)
		msgs <- info
	}()

	// start a client and connect to the server
	conn, _ := net.Dial("tcp", "localhost:1235")
	fmt.Fprintln(conn, "information!")

	info := <-msgs
	if info != "information!" {
		t.Errorf("information not transmitted between server and client")
	}

}

func TestPoolServer(t *testing.T) {

	serv, _ := net.Dial("tcp", "localhost:1234")

	fmt.Fprintln(serv, "pool")
	fmt.Fprintln(serv, "new")

	var serverPort string
	fmt.Fscanln(serv, &serverPort)
	if serverPort != "1236" {
		t.Errorf("Server reported an inaccurate port: %s", serverPort)
	}

	var serverID string
	fmt.Fscanln(serv, &serverID)

	serv2, _ := net.Dial("tcp", "localhost:1234")
	fmt.Fprintln(serv2, "pool")
	fmt.Fprintln(serv2, serverID)

	msgs := make(chan string)
	go func() {
		var info string
		fmt.Fscanln(serv, &info)
		msgs <- info
	}()

	go func() {
		var info string
		fmt.Fscanln(serv2, &info)
		msgs <- info
	}()

	// start a client and connect to the server
	conn, _ := net.Dial("tcp", "localhost:1236")
	fmt.Fprintln(conn, "information!")

	info := <-msgs
	if info != "information!" {
		t.Errorf("information not transmitted between server and client (%s)", info)
	}

}

func TestEventServer(t *testing.T) {

	serv, _ := net.Dial("tcp", "localhost:1234")

	fmt.Fprintln(serv, "event")
	fmt.Fprintln(serv, "new")

	var serverPort string
	fmt.Fscanln(serv, &serverPort)
	if serverPort != "1237" {
		t.Errorf("Server reported an inaccurate port: %s", serverPort)
	}

	newID := make(chan string)
	go func() {
		var id string
		fmt.Fscanln(serv, &id)
		newID <- id
	}()

	// start a client and connect to the server
	conn, _ := net.Dial("tcp", "localhost:1237")

	fmt.Fprintln(conn, "information!")

	serverConnection := <-newID

	serv2, _ := net.Dial("tcp", "localhost:1234")

	fmt.Fprintln(serv2, "event")
	fmt.Fprintln(serv2, serverConnection)

	var info string
	fmt.Fscanln(serv2, &info)

	if info != "information!" {
		t.Errorf("information not transmitted between server and client (%s)", info)
	}

}

func TestForwarding(t *testing.T) {
	os.Args = []string{"relay", "--localForwardPort=3240", "localhost", "1234"}

	flag.Parse()

	go localProxy()
	// fire up the proxy

	go func() {
		listener, _ := net.Listen("tcp", ":3240")
		for {
			conn, _ := listener.Accept()
			io.Copy(conn, conn)
		}		
	}()
	<-time.After(1*time.Millisecond)
	conn, _ := net.Dial("tcp", "localhost:1238")
	fmt.Fprintln(conn, "helloFriend")

	var str string
	fmt.Fscanln(conn, &str)
	if str != "helloFriend" {
		t.Errorf("information not transmitted between server and client (%s)", str)
	}	

}

func BenchmarkThroughput(b *testing.B) {
	serv, _ := net.Dial("tcp", "localhost:1234")

	var serverPort string
	fmt.Fscanln(serv, &serverPort)

	session, _ := yamux.Server(serv, yamux.DefaultConfig())

	go func() {
		for {
			servConn, _ := session.Accept()

			io.Copy(servConn, servConn)

		}
	}()

	// start a client and connect to the server
	conn, _ := net.Dial("tcp", fmt.Sprintf("localhost:%s", serverPort))

	num := 0
	go func() {
		for {
			fmt.Fscanln(conn, &num)
		}
	}()

	for i := 0; i < b.N; i++ {
		fmt.Println(i)
		fmt.Fprintln(conn, i)
	}

}

func BenchmarkConnections(b *testing.B) {
	serv, _ := net.Dial("tcp", "localhost:1234")

	var serverPort string
	fmt.Fscanln(serv, &serverPort)

	session, _ := yamux.Server(serv, yamux.DefaultConfig())

	go func() {
		for {
			servConn, _ := session.Accept()

			io.Copy(servConn, servConn)

		}
	}()

	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine has its own bytes.Buffer.
		for pb.Next() {
			// start a client and connect to the server
			conn, _ := net.Dial("tcp", fmt.Sprintf("localhost:%s", serverPort))

			word := ""
			go func() {
				for {
					fmt.Fscanln(conn, &word)
				}
			}()
			fmt.Fprintln(conn, "hello")
			conn.Close()
		}
	})

}
