.PHONY:	build push run

IMAGE = ghcr.io/xonvanetta/tibber-inflxudb

# supply when running make: make all TAG=1.0.0
#TAG = 0.0.1

build:
	CGO_ENABLED=0 GOOS=linux go build ./cmd/tibber-influxdb

docker: build
	docker build --pull --rm -t $(IMAGE):$(TAG) .
	rm tibber-influxdb

push: docker
	docker push $(IMAGE):$(TAG)

all: build docker push

run:
	docker run -it --env-file .env --rm -p 9501:9501 -t $(IMAGE):$(TAG)

test: fmt
	go test ./...

localrun:
	bash -c "env `grep -Ev '^#' .env | xargs` go run ./cmd/..."
fmt:
	bash -c "test -z $$(gofmt -l $$(find . -type f -name '*.go' -not -path './vendor/*') | tee /dev/stderr) || (echo 'Code not formatted correctly according to gofmt!' && exit 1)"