
VERSION ?= git-$(shell git rev-parse --short=9 HEAD)

build:
	GOOS="linux" go build -o ./cmd/frontend/run ./cmd/frontend
	GOOS="linux" go build -o ./cmd/customers/run ./cmd/customers

	docker build --no-cache -t pskreter/frontend:${VERSION} ./cmd/frontend
	docker build --no-cache -t pskreter/customers:${VERSION} ./cmd/customers

	rm ./cmd/frontend/run
	rm ./cmd/customers/run

run: build
	IMAGE_VERSION=${VERSION} docker-compose up

