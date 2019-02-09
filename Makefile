deps:
	go get -d -t ./...

test: deps
	go test -v

build: deps
	goxz -os=linux -arch=386,amd64 -d=dist -z

lint:
	go vet
	golint -set_exit_status
