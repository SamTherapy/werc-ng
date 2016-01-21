P=gowerc
SRC=bitbucket.org
USER?=mischief
GO_VERSION=1.5.3

all:	$(USER)/$(P)

$(USER)/$(P):	bin/$(P)
	docker build -t "$(USER)/$(P):latest" .

bin/$(P):	*.go
	docker run --rm -v ${PWD}:/go/src/$(SRC)/$(USER)/$(P) golang:${GO_VERSION} /bin/bash -c "go list -f '{{range .Imports}}{{printf \"%s\n\" .}}{{end}}' $(SRC)/$(USER)/$(P) | xargs go get -d; CGO_ENABLED=0 go build -v -installsuffix cgo -o /go/src/$(SRC)/$(USER)/$(P)/bin/$(P) $(SRC)/$(USER)/$(P)"

clean:
	docker run --rm -v ${PWD}:/opt busybox rm -f /opt/bin/$(P)
	docker rmi -f "$(USER)/$(P):latest"

