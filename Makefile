
test:
	go test -v ./...

install:
	go build -o ${GOBIN}/templar ./cmd/templar/*.go
