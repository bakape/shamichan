.PHONY: all clean

all: clean
	$(GOPATH)/bin/go-bindata -o sql/sql.go --pkg sql --nometadata --prefix sql sql/...

clean:
	rm -f sql/sql.go
