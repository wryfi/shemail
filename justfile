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
    rm -rf release/*

install:
    go build -o {{ BINDIR }}/shemail -ldflags "{{ LDFLAGS }}"

release: clean releasedir
    GOARCH=arm64 GOOS=darwin go build -o release/shemail_{{ VERSION }}_darwin_arm64 -ldflags "{{ LDFLAGS }}" .
    pushd release && shasum -a 256 shemail_{{ VERSION }}_darwin_arm64 >> sha256sums_{{ VERSION }}.txt && popd
    pushd release && mv shemail_{{ VERSION }}_darwin_arm64 shemail && popd
    pushd release && zip shemail_{{ VERSION }}_darwin_arm64.zip shemail && rm shemail && popd

    GOARCH=amd64 GOOS=darwin go build -o release/shemail_{{ VERSION }}_darwin_amd64 -ldflags "{{ LDFLAGS }}" .
    pushd release && shasum -a 256 shemail_{{ VERSION }}_darwin_amd64 >> sha256sums_{{ VERSION }}.txt && popd
    pushd release && mv shemail_{{ VERSION }}_darwin_amd64 shemail && popd
    pushd release && zip shemail_{{ VERSION }}_darwin_amd64.zip shemail && rm shemail && popd

    GOARCH=arm64 GOOS=linux go build -o release/shemail_{{ VERSION }}_linux_arm64 -ldflags "{{ LDFLAGS }}" .
    pushd release && shasum -a 256 shemail_{{ VERSION }}_linux_arm64 >> sha256sums_{{ VERSION }}.txt && popd
    pushd release && mv shemail_{{ VERSION }}_linux_arm64 shemail && popd
    pushd release && zip shemail_{{ VERSION }}_linux_arm64.zip shemail && rm shemail && popd

    GOARCH=amd64 GOOS=linux go build -o release/shemail_{{ VERSION }}_linux_amd64 -ldflags "{{ LDFLAGS }}" .
    pushd release && shasum -a 256 shemail_{{ VERSION }}_linux_amd64 >> sha256sums_{{ VERSION }}.txt && popd
    pushd release && mv shemail_{{ VERSION }}_linux_amd64 shemail && popd
    pushd release && zip shemail_{{ VERSION }}_linux_amd64.zip shemail && rm shemail && popd

    GOARCH=arm64 GOOS=windows go build -o release/shemail_{{ VERSION }}_windows_arm64 -ldflags "{{ LDFLAGS }}" .
    pushd release && shasum -a 256 shemail_{{ VERSION }}_windows_arm64 >> sha256sums_{{ VERSION }}.txt && popd
    pushd release && mv shemail_{{ VERSION }}_windows_arm64 shemail && popd
    pushd release && zip shemail_{{ VERSION }}_windows_arm64.zip shemail && rm shemail && popd

    GOARCH=amd64 GOOS=windows go build -o release/shemail_{{ VERSION }}_windows_amd64 -ldflags "{{ LDFLAGS }}" .
    pushd release && shasum -a 256 shemail_{{ VERSION }}_windows_amd64 >> sha256sums_{{ VERSION }}.txt && popd
    pushd release && mv shemail_{{ VERSION }}_windows_amd64 shemail && popd
    pushd release && zip shemail_{{ VERSION }}_windows_amd64.zip shemail && rm shemail && popd

releasedir:
    mkdir -p release

run *FLAGS:
    go run -ldflags "{{ LDFLAGS }}" . {{FLAGS}}

test:
    go test -race -v ./...