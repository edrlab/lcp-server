ROOT_DIR=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

BUILD_DIR=$(ROOT_DIR)/build

UNAME_S= $(shell uname -s)
ARCH=amd64

lcpchecker=cmd/lcpchecker/lcpchecker.go

#LDFLAGS=-ldflags '-linkmode external -w -extldflags "-static"'
LDFLAGS=

rm=rm -rf

#amd64
CC=go install -x  $(LDFLAGS)

.PHONY: all clean $(lcpchecker)

all: $(lcpchecker)

clean:
	$(rm) $(BUILD_DIR)

$(lcpchecker):
	GOPATH=$(BUILD_DIR) GOARCH=$(ARCH) $(CC) ./$@
