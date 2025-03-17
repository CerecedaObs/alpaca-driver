

all: alpaca-driver.exe

alpaca-driver.exe:
	GOOS=windows GOARCH=amd64 go build -o alpaca-driver.exe ./cmd/main.go
