all:
	go build -o fhomectl main.go env.go

clean:
	rm ./fhomectl
