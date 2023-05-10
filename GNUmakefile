.PHONY: \
	all \
	build \
	install \
	lint \
	golangci \
	vet \
	fmt \
	fmtcheck \
	pretest \
	test \
	integration \
	cov \
	clean \
	build-integration \
	clean-integration \
	fetch-rdb \
	fetch-redis \
	diff-server-rdb \
	diff-server-redis \
	diff-server-rdb-redis

SRCS = $(shell git ls-files '*.go')
PKGS = $(shell go list ./...)
VERSION := $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)
LDFLAGS := -X 'github.com/vulsio/go-cpe-dictionary/config.Version=$(VERSION)' \
	-X 'github.com/vulsio/go-cpe-dictionary/config.Revision=$(REVISION)'
GO := CGO_ENABLED=0 go

all: build test

build: main.go
	$(GO) build -ldflags "$(LDFLAGS)" -o go-cpe-dictionary $<

install: main.go
	$(GO) install -ldflags "$(LDFLAGS)"

lint:
	go install github.com/mgechev/revive@latest
	revive -config ./.revive.toml -formatter plain $(PKGS)

golangci:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

vet:
	echo $(PKGS) | xargs env $(GO) vet || exit;

fmt:
	gofmt -w $(SRCS)

fmtcheck:
	$(foreach file,$(SRCS),gofmt -d $(file);)

pretest: lint vet fmtcheck

test: pretest
	$(GO) test -cover -v ./... || exit;

cov:
	@ go get -v github.com/axw/gocov/gocov
	@ go get golang.org/x/tools/cmd/cover
	gocov test | gocov report

clean:
	$(foreach pkg,$(PKGS),go clean $(pkg) || exit;)

PWD := $(shell pwd)
BRANCH := $(shell git symbolic-ref --short HEAD)
build-integration:
	@ git stash save
	$(GO) build -ldflags "$(LDFLAGS)" -o integration/go-cpe.new
	git checkout $(shell git describe --tags --abbrev=0)
	@git reset --hard
	$(GO) build -ldflags "$(LDFLAGS)" -o integration/go-cpe.old
	git checkout $(BRANCH)
	-@ git stash apply stash@{0} && git stash drop stash@{0}

clean-integration:
	-pkill go-cpe.old
	-pkill go-cpe.new
	-rm integration/go-cpe.old integration/go-cpe.new integration/go-cpe.old.sqlite3 integration/go-cpe.new.sqlite3
	-rm -rf integration/diff
	-docker kill redis-old redis-new
	-docker rm redis-old redis-new

fetch-rdb:
	integration/go-cpe.old fetch nvd --dbpath=$(PWD)/integration/go-cpe.old.sqlite3
	integration/go-cpe.old fetch jvn --dbpath=$(PWD)/integration/go-cpe.old.sqlite3
	integration/go-cpe.new fetch nvd --dbpath=$(PWD)/integration/go-cpe.new.sqlite3
	integration/go-cpe.new fetch jvn --dbpath=$(PWD)/integration/go-cpe.new.sqlite3

fetch-redis:
	docker run --name redis-old -d -p 127.0.0.1:6379:6379 redis
	docker run --name redis-new -d -p 127.0.0.1:6380:6379 redis

	integration/go-cpe.old fetch nvd --dbtype redis --dbpath "redis://127.0.0.1:6379/0"
	integration/go-cpe.old fetch jvn --dbtype redis --dbpath "redis://127.0.0.1:6379/0"
	integration/go-cpe.new fetch nvd --dbtype redis --dbpath "redis://127.0.0.1:6380/0"
	integration/go-cpe.new fetch jvn --dbtype redis --dbpath "redis://127.0.0.1:6380/0"

diff-server-rdb:
	integration/go-cpe.old server --dbpath=$(PWD)/integration/go-cpe.old.sqlite3 --port 1325 > /dev/null 2>&1 & 
	integration/go-cpe.new server --dbpath=$(PWD)/integration/go-cpe.new.sqlite3 --port 1326 > /dev/null 2>&1 &
	@ python integration/diff_server_mode.py cpes --sample_rate 0.01
	pkill go-cpe.old 
	pkill go-cpe.new

diff-server-redis:
	integration/go-cpe.old server --dbtype redis --dbpath "redis://127.0.0.1:6379/0" --port 1325 > /dev/null 2>&1 &
	integration/go-cpe.new server --dbtype redis --dbpath "redis://127.0.0.1:6380/0" --port 1326 > /dev/null 2>&1 &
	@ python integration/diff_server_mode.py cpes --sample_rate 0.01
	pkill go-cpe.old 
	pkill go-cpe.new

diff-server-rdb-redis:
	integration/go-cpe.new server --dbpath=$(PWD)/integration/go-cpe.new.sqlite3 --port 1325 > /dev/null 2>&1 &
	integration/go-cpe.new server --dbtype redis --dbpath "redis://127.0.0.1:6380/0" --port 1326 > /dev/null 2>&1 &
	@ python integration/diff_server_mode.py cpes --sample_rate 0.01
	pkill go-cpe.new
