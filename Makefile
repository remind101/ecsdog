bin/ecsdog: *.go
	go build -o $@ ./cmd/ecsdog

bin/ecsdog-linux: *.go
	GOOS=linux go build -o $@ ./cmd/ecsdog

docker::
	docker build -t remind101/ecsdog .
