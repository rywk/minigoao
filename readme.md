# Minigoao

## Example run
In 2 different terminals.

Server: `./server.sh 5555 false`

The server command should output an address with port (`127.0.0.1:5555`), copy that so you can login the client.

Client: `./game.sh`


### How to run the server
`./server.sh <port> <expose>`.
- `<port>`
- `<expose>`
  - `true` is `0.0.0.0`, server is exposed to default routing, can be reached from the internet.
  - `false` is `127.0.0.1`, server is only exposed locally, can't be reached from the internet.


### How to run the client

`./game.sh`

### How to build the client

`./build_game.sh`, should output `minigoao.exe`.
