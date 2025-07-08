all: run

build:
	go build .
run:
	go run main.go
clean: 
	rm swiggy-cli

