# This Makefile delegates to the repository-root Makefile.
# Usage: cd app/<service> && make build   (or make run, test, etc.)
ROOT := $(shell git rev-parse --show-toplevel 2>/dev/null || echo ../..)

.PHONY: api config build run test tidy proto generate
api:
	cd $(ROOT) && make api
config:
	cd $(ROOT) && make config
build:
	cd $(ROOT) && go build ./app/$(notdir $(CURDIR))/...
run:
	cd $(ROOT) && go run ./app/$(notdir $(CURDIR))/cmd/$(notdir $(CURDIR))
test:
	cd $(ROOT) && go test ./app/$(notdir $(CURDIR))/...
tidy:
	cd $(ROOT) && go mod tidy
proto:
	cd $(ROOT) && make proto
generate:
	cd $(ROOT) && make generate
