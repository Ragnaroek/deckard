build:
	CGO_ENABLED=1 go build -o deckard cmd/main.go

run: build
	./deckard