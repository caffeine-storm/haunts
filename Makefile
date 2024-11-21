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

debug: devhaunts
	./$^

dlv: devhaunts ${RUNTIME_DATADIR}
	dlv debug --build-flags='-modfile dev.go.mod -tags nosound' .

devhaunts:
	go build -x -modfile dev.go.mod -o devhaunts -tags nosound main.go GEN_version.go

haunts: GEN_version.go
	go build -x -o haunts -tags nosound main.go GEN_version.go

# TODO(tmckee): this should use 'go gen' instead
GEN_version.go: tools/version.go .git/HEAD
	go run -C tools version.go

${RUNTIME_DATADIR}: ${SRC_DATADIR}
	cp -r $^ $@

clean:
	rm -rf ${RUNTIME_DATADIR}

fmt:
	go fmt ./...

# -l for 'list files'
checkfmt:
	@gofmt -l ./

lint:
	go run github.com/mgechev/revive@v1.5.1 ./...

test:
	go test ${testrunargs}                     -tags nosound ./...

test-nocache:
	go test ${testrunargs} -count=1            -tags nosound ./...

test-dlv: singlepackage=${pkg}
test-dlv:
	dlv test --build-flags="-tags nosound" ${singlepackage} -- ${testrunargs}

devtest:
	go test ${testrunargs} -modfile dev.go.mod -tags nosound ./...

update-glop:
	go -C tools/update-glop/ run cmd/main.go
	go mod tidy

update-appveyor-image:
	go run tools/update-appveyor-image/main.go ./appveyor.yml

# Let go tooling decide if things are out-of-date
.PHONY: haunts
.PHONY: devhaunts
.PHONY: clean
.PHONY: fmt lint
.PHONY: test devtest
.PHONY: update-glop update-appveyor-image
