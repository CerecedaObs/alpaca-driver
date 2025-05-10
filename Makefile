

all: zro-alpaca.exe

test:
	go test ./...

zro-alpaca.exe:
	GOOS=windows GOARCH=amd64 go build ./cmd/zro-alpaca

clean:
	rm -f zro-alpaca.exe


.PHONY: all clean