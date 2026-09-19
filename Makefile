APP_NAME=api
MAIN=./cmd/api

run:
	go run $(MAIN)

run-dummy-his:
	go run ./cmd/his-a-dummy

build:
	go build -o bin/$(APP_NAME) $(MAIN)

test:
	go test ./...
