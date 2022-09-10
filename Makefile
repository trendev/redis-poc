run: clean build
	./bin/client
build: clean
	go build -o bin/client -v cmd/main.go
clean:
	rm -rf ./bin
test:
	go clean -testcache
	go test -v ./... 