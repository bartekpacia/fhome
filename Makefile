all: fh fhomed

fh: cmd/fh/main.go
	go build -o fh cmd/fh/main.go

fhomed: cmd/fhomed/main.go
	go build -o fhomed cmd/fhomed/main.go

clean:
	rm -f ./fh ./fhomed
