.PHONY: \
	build \
	install \
	all \
	vendor \
	lint \
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
PKGS =  ./config ./db ./models
VERSION := $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)
LDFLAGS := -X 'github.com/kotakanbe/go-cpe-dictionary/config.Version=$(VERSION)' \
	-X 'github.com/kotakanbe/go-cpe-dictionary/config.Revision=$(REVISION)'
GO := GO111MODULE=on go
GO_OFF := GO111MODULE=off go

all: build test

build: main.go
	go build -ldflags "$(LDFLAGS)" -o go-cpe-dictionary $<

install: main.go
	go install -ldflags "$(LDFLAGS)"

all: test

lint:
	@ go get -u golang.org/x/lint/golint
	$(foreach file,$(SRCS),golint $(file) || exit;)

vet:
	$(foreach pkg,$(PKGS),go vet $(pkg);)

fmt:
	gofmt -w $(SRCS)

fmtcheck:
	$(foreach file,$(SRCS),gofmt -d $(file);)

pretest: lint vet fmtcheck

test: pretest
	$(foreach pkg,$(PKGS),go test -v $(pkg) || exit;)

integration:
	go test -tags docker_integration -run TestIntegration -v

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
	# git checkout $(shell git describe --tags --abbrev=0)
	git checkout upstream/master
	@git reset --hard
	$(GO) build -ldflags "$(LDFLAGS)" -o integration/go-cpe.old
	git checkout $(BRANCH)
	-@ git stash apply stash@{0} && git stash drop stash@{0}

clean-integration:
	-pkill go-cpe.old
	-pkill go-cpe.new
	-rm integration/go-cpe.old integration/go-cpe.new integration/go-cpe.old.sqlite3 integration/go-cpe.new.sqlite3
	-docker kill redis-old redis-new
	-docker rm redis-old redis-new

fetch-rdb:
	integration/go-cpe.old fetchnvd --dbpath=$(PWD)/integration/go-cpe.old.sqlite3
	integration/go-cpe.old fetchjvn --dbpath=$(PWD)/integration/go-cpe.old.sqlite3
	integration/go-cpe.new fetchnvd --dbpath=$(PWD)/integration/go-cpe.new.sqlite3
	integration/go-cpe.new fetchjvn --dbpath=$(PWD)/integration/go-cpe.new.sqlite3

fetch-redis:
	docker run --name redis-old -d -p 127.0.0.1:6379:6379 redis
	docker run --name redis-new -d -p 127.0.0.1:6380:6379 redis

	integration/go-cpe.old fetchnvd --dbtype redis --dbpath "redis://127.0.0.1:6379/0"
	integration/go-cpe.old fetchjvn --dbtype redis --dbpath "redis://127.0.0.1:6379/0"
	integration/go-cpe.new fetchnvd --dbtype redis --dbpath "redis://127.0.0.1:6380/0"
	integration/go-cpe.new fetchjvn --dbtype redis --dbpath "redis://127.0.0.1:6380/0"

diff-server-rdb:
	integration/go-cpe.old server --dbpath=$(PWD)/integration/go-cpe.old.sqlite3 --port 1325 > /dev/null & 
	integration/go-cpe.new server --dbpath=$(PWD)/integration/go-cpe.new.sqlite3 --port 1326 > /dev/null &
	@ python integration/diff_server_mode.py cpes
	pkill go-cpe.old 
	pkill go-cpe.new

diff-server-redis:
	integration/go-cpe.old server --dbtype redis --dbpath "redis://127.0.0.1:6379/0" --port 1325 > /dev/null & 
	integration/go-cpe.new server --dbtype redis --dbpath "redis://127.0.0.1:6380/0" --port 1326 > /dev/null &
	@ python integration/diff_server_mode.py cpes
	pkill go-cpe.old 
	pkill go-cpe.new

diff-server-rdb-redis:
	integration/go-cpe.new server --dbpath=$(PWD)/integration/go-cpe.new.sqlite3 --port 1325 > /dev/null &
	integration/go-cpe.new server --dbtype redis --dbpath "redis://127.0.0.1:6380/0" --port 1326 > /dev/null &
	@ python integration/diff_server_mode.py cpes
	pkill go-cpe.new