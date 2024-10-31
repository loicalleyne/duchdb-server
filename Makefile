.PHONY: update_libchdb all test clean

update_libchdb:
	./update_libchdb.sh

install:
	curl -sL https://lib.chdb.io | bash

test:
	CGO_ENABLED=1 go test -v -coverprofile=coverage.out ./...

run:
	# add ld path for linux and macos
	LD_LIBRARY_PATH=/usr/local/lib DYLD_LIBRARY_PATH=/usr/local/lib CGO_ENABLED=1 go run -ldflags '-extldflags "-Wl,-rpath,/usr/local/lib"' main.go duck.go ch.go

mod:
	go get github.com/marcboeker/go-duckdb@latest
	go get github.com/loicalleyne/chdbsession

build-mod:
	LD_LIBRARY_PATH=/usr/local/lib DYLD_LIBRARY_PATH=/usr/local/lib  CGO_ENABLED=1 go build -mod=readonly -ldflags '-extldflags "-Wl,-rpath,/usr/local/lib"' -o duchdb-server-mod 

build-mac:
	go install github.com/goware/modvendor@latest
	go work vendor
	modvendor -copy="**/*.a **/*.h" -v
	LD_LIBRARY_PATH=/usr/local/lib DYLD_LIBRARY_PATH=/usr/local/lib  CGO_ENABLED=1 go build -mod=vendor -ldflags '-extldflags "-Wl,-rpath,/usr/local/lib"' -o duchdb-server 
	install_name_tool -change libchdb.so /usr/local/lib/libchdb.so duchdb-server

build:
	go install github.com/goware/modvendor@latest
	go work vendor
	modvendor -copy="**/*.a **/*.h" -v
	LD_LIBRARY_PATH=/usr/local/lib DYLD_LIBRARY_PATH=/usr/local/lib  CGO_ENABLED=1 go build -mod=vendor -ldflags '-extldflags "-Wl,-rpath,/usr/local/lib"' -o duchdb-server 