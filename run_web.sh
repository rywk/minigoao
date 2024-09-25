#GOOS=windows GOARCH=amd64 go build -o ./cmd/web-server/miniao.exe ./cmd/run-client
GOOS=js GOARCH=wasm go build -o ./cmd/web-server/main.wasm ./cmd/run-web-client

go run ./cmd/web-server -http $1
