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
    GOARCH=arm64 GOOS=darwin go build -o build/rwx_{{ VERSION }}_darwin_arm64 -ldflags "{{ LDFLAGS }}" .

run *FLAGS:
    go run -ldflags "{{ LDFLAGS }}" . {{FLAGS}}

test:
    go test -v ./...