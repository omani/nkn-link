build:
	go build -o nkn-link -ldflags="-w -s" main.go

.PHONY: build