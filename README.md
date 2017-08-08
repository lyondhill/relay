
## Relay

Relay allows non communicative servers to communicate with each other through an intermediary

## How It Works

Relay works by establishing a TCP listener that waits for connections to come in. Once a connection
is established relay will attempt to establish a new TCP listener to facilitate this relay client's connections.
Relay will then report to the relay client the port it has available for communication. Relay supports three protocols for client connections.

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

  There are three possible means of integrating relay into your server.

    1 You can use a multiplexer to send multiple connections through a single TCP connection.
    2 You can create a connection pool that will accept connections for this one request chain.
    3 you can wait for connection events before establishing a new connection. 

  With all three options the potocol starts the same. Your program will connect to the relay and indicate which of the 3 protocols you have implemented. Your program will do this by sending a plain text message of `multiplex\n` for multiplexing, `pool\n` for a connection pool, or `event\n` for event listener.

#### For Multiplexing

  Once your program has specified multiplexing, relay will send a plain text string followed by a new line character. The string is the port it was able to establish for you to use to establish new connections to the relay server. For example if the server is able to establish a listener on port 4532 the first thing the server would send back would be `4532\n`. At this point the next thing that needs to be done is to escalate the tcp connection to a multiplexed server. The relay client is a server so in the multiplexed connection it establishes itself as a server. The specific multiplexing protocol relay uses is defined by [yamux](https://github.com/hashicorp/yamux). From this point on you will wait to accept connections as you usually would.

  Communication Series

| Client               | Server        | user                  |
| -------------------- | ------------- | --------------------- |
| multiplex\n          |               |                       |
|                      | [port]\n      |                       |
| wait for connections |               |                       |
|                      |               |  establish connection |
|                      | <- pipe ->    |                       |


#### For Connection Pools

  Now that you have selected the `pool` protocol you now have to tell the server if your a new or existing pool connection. To establish a new relay you will simply send `new\n` to the server. When a new connection is established the server will indicate that it is ready to recieve new connections for your designated connection pool with `<port_number>\n` followed by `<id>\n`. As soon as you recieve the id from the server you now will have 1 connection waiting in the connection pool. Once a connection is needed data will begin flowing through the connection, When it is done being used it will disconnect. To add additional connections to your connection pool, you simply establish a new connection to the relay server and pass `pool\n` followed by `<id>\n` where the id is the id provided by the server in your first attempt.

  Communication Series

| Client               | Server        | user                  |
| -------------------- | ------------- | --------------------- |
| pool\n               |               |                       |
| new\n                |               |                       |
|                      | [port]\n      |                       |
|                      | [id]\n        |                       |
| wait for connections |               |                       |
|                      |               |  establish connection |
|                      | <- pipe ->    |                       |


#### For Event listeners

  Now that you have selected `event` protocol you need to indicate if your establishing a new event listener or your are responding to an event. To establish a new event listener you simply send `new\n` to the server. When this happens the server will confirm your ready to recieve events by sending `<port>\n` back to the client. When a new connection for your event is establisht with the server the server will send your event client an id that you will use to recieve the connection. the connection on the wire will look 

  Communication Series

| Client                | Server        | user                  |
| --------------------- | ------------- | --------------------- |
| event\n               |               |                       |
| new\n                 |               |                       |
|                       | [port]\n      |                       |
| wait for connections  |               |                       |
|                       |               |  establish connection |
|                       | [id]\n        |                       |
| create new connection |               |                       |
| event\n               |               |                       |
| [id]\n                |               |                       |
|                       | <- pipe ->    |                       |



