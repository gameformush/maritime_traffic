build:
	go build -o traffic cmd/main.go
	
test:
	go test -v ./...
    
watch:
	while true; do \
	$(MAKE) test; \
	fswatch -1 -e ".*" -i "\\.go$$" .; \
	done