#!/bin/sh
mkdir -p bin in out
cd binsrc/go
go build -o ../../bin/go .
cd ../..
g++ `pkg-config --libs --cflags vips-cpp` \
	binsrc/vips/vips.cc -o bin/vips -O2
g++ `pkg-config --libs --cflags Magick++` \
	binsrc/magick/magick.cc -o bin/magick -O2

go build .
