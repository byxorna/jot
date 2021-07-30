NAME := jot
git_commit := $(shell git rev-parse HEAD)
git_tag := $(shell git describe --tags --always HEAD)
date := $(shell date)
pkg := $(shell go list -m)

all: build

.PHONY: build dev

build:
	@go build -o bin/$(NAME) \
		-ldflags "-X '$(pkg)/pkg/version.Commit=$(git_commit)' -X '$(pkg)/pkg/version.Date=$(date)' -X '$(pkg)/pkg/version.Version=$(git_tag)'" ./

test: build
	go test -v ./...

dev: build
	@bin/jot --use-alt-screen=false
	
install: build
	cp bin/$(NAME) ~/bin/$(NAME)

pprof-heap: build
	go tool pprof http://localhost:6060/debug/pprof/heap

pprof-allocs: build
	go tool pprof http://localhost:6060/debug/pprof/allocs

pprof-profile: build
	curl -o profile.pprof http://localhost:6060/debug/pprof/profile?seconds=5
	go tool pprof profile.pprof
