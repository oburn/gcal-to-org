.PHONY: help tidy test tui run build
.DEFAULT_GOAL := help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

upgrade-list:	## List available upgrades
	go list -u -m all

upgrade-perform:	## Perform upgrade of dependencies
	go get -u ./...
	go get -t -u ./...

tidy:	## Tidy up the modules
	go mod tidy

test:	## Run all tests
	go test -v

run:	## Run app using arguments specified with `make run ARGS="a b c"
	go run . $(ARGS)

build: gcal-to-org	## Build the binary file

gcal-to-org: main.go secrets.go
	go build .
