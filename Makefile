all:
	go build cmd/main.go

clean: main.go
	rm main
