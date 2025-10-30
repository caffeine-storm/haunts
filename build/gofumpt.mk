gofmt:
	go run mvdan.cc/gofumpt@v0.9.2 -l -w .

checkgofmt:
	@go run mvdan.cc/gofumpt@v0.9.2 -l .

.PHONY: gofmt checkgofmt
