GO=	env GOPATH=`pwd` go

all: hugopix

DEPS=	src/github.com/artyom/smartcrop \
	src/github.com/nfnt/resize \
	src/github.com/rwcarlsen/goexif/exif \
	src/github.com/termie/go-shutil

hugopix: $(DEPS) hugopix.go
	$(GO) build hugopix.go

src/github.com/artyom/smartcrop:
	$(GO) get -f -t -u -v github.com/artyom/smartcrop

src/github.com/nfnt/resize:
	$(GO) get -f -t -u -v github.com/nfnt/resize

src/github.com/rwcarlsen/goexif/exif:
	$(GO) get -f -t -u -v github.com/rwcarlsen/goexif/exif

src/github.com/termie/go-shutil:
	$(GO) get -f -t -u -v github.com/termie/go-shutil

test:
	$(GO) test

clean:
	-rm -rf src pkg hugopix *~ core
