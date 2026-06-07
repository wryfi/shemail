APP := "shemail"
HOME := env_var("HOME")
BINDIR := HOME + "/.local/bin"

MODULE := "github.com/wryfi/shemail"
VERSION := `git describe --tags --dirty 2> /dev/null || echo v0`
REVISION := `git rev-parse --short HEAD 2> /dev/null || echo 0`
BUILD_DATE := `date -u +'%FT%T%:z'`

LDFLAG_VERSION := "-X github.com/wryfi/shemail/config.ShemailVersion=" + VERSION
LDFLAG_REVISION := "-X github.com/wryfi/shemail/config.GitRevision=" + REVISION
LDFLAG_BUILD_DATE := "-X github.com/wryfi/shemail/config.BuildDate=" + BUILD_DATE

LDFLAGS :=  LDFLAG_VERSION + " " + LDFLAG_REVISION + " " + LDFLAG_BUILD_DATE + " -w -s"

default:
    @just --list --justfile {{ justfile() }}

build:
    go build -o build/{{ APP }} -race -ldflags "{{ LDFLAGS }}"

clean:
    rm -rf build/*
    rm -rf dist/

install:
    go build -o {{ BINDIR }}/shemail -ldflags "{{ LDFLAGS }}"

# build all release artifacts locally into dist/ without publishing
snapshot:
    goreleaser release --snapshot --clean

# build and publish a release (needs a pushed tag + GITHUB_TOKEN; usually CI does this)
release:
    goreleaser release --clean

run *FLAGS:
    go run -ldflags "{{ LDFLAGS }}" . {{FLAGS}}

test:
    go test -race -v ./...