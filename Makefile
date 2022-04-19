all:
	go build -o fhome main.go messages.go 

cli:
	go build -o fhome cmd/main.go

clean:
	rm fhome
