all: clean
	$(GOPATH)/bin/go-bindata -o bin.go --pkg imager --nocompress --nometadata archive.png audio.png

clean:
	rm -f bin.go
