all:
	go build -o fh cmd/main.go cmd/env.go

clean: main.go
	rm main
