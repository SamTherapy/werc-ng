GO = go

CGO_ENABLED = 0

all: go2werc

go2werc:
	$GO build -ldflags="-s -w" .

fmt:
	gofmt -w -s .

vet:
	$GO vet ./...

lint: fmt vet
	-golangci-lint run

clean:
	$GO clean