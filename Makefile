.PHONY: test build vet lint wasm cardart clean

test:
	go test ./...

build:
	go build ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...
	staticcheck ./...

wasm:
	GOOS=js GOARCH=wasm go build -o web/mage.wasm ./cmd/wasm/
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/

cardart:
	go build -o cardart ./cmd/cardart/

clean:
	go clean ./...
	rm -f tui server cardgen fetchset genset cardart
	rm -f coverage.html coverage.out *.coverprofile
	rm -f web/mage.wasm web/wasm_exec.js
