.POSIX:
.SUFFIXES:

GO?=go
CXX?=g++

gifbench: main.go go.mod go.sum
	$(GO) build -o $@ .

bin/go: binsrc/go/main.go go.mod go.sum
	$(GO) build -o $@ ./binsrc/go

bin/vips: binsrc/vips/vips.cc
	$(CXX) `pkg-config --libs --cflags vips-cpp` \
	binsrc/vips/vips.cc -o $@ -O2

bin/magick: binsrc/magick/magick.cc
	$(CXX) `pkg-config --libs --cflags Magick++` \
	binsrc/magick/magick.cc -o $@ -O2

BINS := \
	bin/go \
	bin/magick \
	bin/vips

all: gifbench $(BINS)

RM?=rm -f

clean:
	$(RM) gifbench $(BINS)

.DEFAULT_GOAL := all

.PHONY: all clean
