build:
	go build -o traffic cmd/main.go
	
test:
	go test -v ./...