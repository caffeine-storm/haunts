ifneq "${testrun}" ""
testrunargs:=-run ${testrun}
else
testrunargs:=
endif

SRC_DATADIR:=data
RUNTIME_DATADIR:=data-runtime

all: haunts

it: haunts
go: haunts ${RUNTIME_DATADIR}
	./haunts

dlv: devhaunts
	LD_LIBRARY_PATH=./lib/linux/ dlv debug --build-flags='-modfile dev.go.mod -tags nosound' .

devhaunts:
	go build -x -modfile dev.go.mod -o devhaunts -tags nosound -ldflags "-extldflags \"-L ./lib/linux/ -Wl,-rpath,\$${ORIGIN}/lib/linux\"" main.go GEN_version.go

haunts: lib/linux/libglop.so GEN_version.go
	go build -x -o haunts -tags nosound -ldflags "-extldflags \"-L ./lib/linux/ -Wl,-rpath,\$${ORIGIN}/lib/linux\"" main.go GEN_version.go

# TODO(tmckee): this should use 'go gen' instead
GEN_version.go: tools/version.go .git/HEAD
	go run -C tools version.go

# TODO(tmckee): add a rule to make lib/linux/libglop.so
# For now, we at least leave a note
lib/linux/libglop.so:
	@echo please build libglop.so manually and put it in ./lib/linux/
	@echo check go.mod to find the correct version of glop
	false

${RUNTIME_DATADIR}: ${SRC_DATADIR}
	cp -r $^ $@

clean:
	rm -rf ${RUNTIME_DATADIR}

fmt:
	go fmt ./...

test:
	go test ${testrunargs} -tags nosound ./...

devtest:
	go test ${testrunargs} -modfile dev.go.mod -tags nosound ./...

# Let go tooling decide if things are out-of-date
.PHONY: haunts
.PHONY: devhaunts
.PHONY: clean
.PHONY: fmt
.PHONY: test
.PHONY: devtest
