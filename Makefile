.PHONY: test build vet lint wasm cardart clean

test:
	go test ./...

build:
	go build ./...

lint:
	golangci-lint fmt ./...
	golangci-lint run --fix ./...
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix ./...

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
