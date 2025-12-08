
test:
	go test ./...

install:
	go build -o ${GOBIN}/templar ./cmd/templar/*.go
