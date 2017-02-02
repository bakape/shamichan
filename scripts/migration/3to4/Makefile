bin=3to4
ifeq ($(OS), Windows_NT)
	bin=3to4.exe
endif

upgrade:
	go build -v .
	./$(bin)

clean:
	rm -f $(bin)
