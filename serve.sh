GOOS=windows GOARCH=amd64 go build -o ./bin/miniao.exe ./cmd/run-client
GOOS=js GOARCH=wasm go build -o ./bin/main.wasm ./cmd/run-web-client
go run ./cmd/run-server $1 $2 $3