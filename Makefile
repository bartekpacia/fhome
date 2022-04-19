all:
	go build -o fhome cmd/main.go cmd/env.go

clean: main.go
	rm main
