.PHONY: test build vet lint cardart clean

test:
	go test ./...

build:
	go build ./...

lint:
	modernize -fix ./...
	golangci-lint fmt ./...
	golangci-lint run --fix ./...

cardart:
	go build -o cardart ./cmd/cardart/

clean:
	go clean ./...
	rm -f tui server cardgen fetchset genset cardart
	rm -f coverage.html coverage.out *.coverprofile
