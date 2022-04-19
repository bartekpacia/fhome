all:
	go build -o fhome main.go messages.go 

clean:
	rm fhome
