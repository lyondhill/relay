
## Relay

Relay allows non communicative servers to communicate with each other through an intermediary

## How It Works

Relay works by establishing a TCP listener that waits for connections to come in. Once a connection
is established relay will attempt to establish a new TCP listener to facilitate this relay client's connections.
Relay will then report to the relay client the port it has available for communication. At this point Both the 
relay server and the relay client switch their standard tcp connection to a (TCP multiplexer)[https://en.wikipedia.org/wiki/Multiplexing]. From this point on any connections that are established on the new TCP listener will create a new multiplex'ed connection to the relay client and forward all traffic through the server to the relay client.

## How to use it

The relay binary can be used in one of two ways:
  - as a relay server using the command `relay <localport>`
  - as a relay forwarder with `relay --localForwardPort=<local_server_port> <host> <port>`

## How to integrate

### User relay as a forwarder

  If your using a server that you dont want to modify, or you are just not interested in making any changes to it, you can initiate the server to listen on localhost, then Start the relay as a forwarder on the same machine.

  lets use redis as an example:

  start redis with `redis-server --port 6379 --bind localhost`
  start relay  with `relay --localForwardPort=6379 <relay_host> <relay_port>`

### Integrate relay into your server
  
  Relays communication is simple. First you establish a connection to the relay server. Once the connection is established relay will send a plain text string followed by a new line character. The string is the port it was able to establish for you to use to establish new connections to the relay server. For example if the server is able to establish a listener on port 4532 the first thing the server would send back would be `4532\n`. At this point the next thing that needs to be done is to escalate the tcp connection to a multiplexed server. The relay client is a server so in the multiplexed connection it establishes itself as a server. The specific multiplexing protocol relay uses is defined by (yamux)[https://github.com/hashicorp/yamux].