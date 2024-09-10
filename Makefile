SRC_DATADIR:=data
RUNTIME_DATADIR:=data-runtime

all: haunts

it: haunts
go: haunts ${RUNTIME_DATADIR}
	./haunts

haunts: lib/linux/libglop.so GEN_version.go
	go build -x -o haunts -tags nosound -ldflags "-extldflags \"-L ./lib/linux/ -Wl,-rpath,\$${ORIGIN}/lib/linux\"" main.go GEN_version.go

# TODO(tmckee): this should use 'go gen' instead
GEN_version.go: tools/version.go .git/HEAD
	go run -C tools version.go

${RUNTIME_DATADIR}: ${SRC_DATADIR}
	cp -r $^ $@

clean:
	rm -rf ${RUNTIME_DATADIR}

# Let go tooling decide if things are out-of-date
.PHONY: haunts

.PHONY: clean
