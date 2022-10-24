run:
	@go run cmd/faucet/main.go

build:
	@go build -o bin/faucet cmd/faucet/main.go