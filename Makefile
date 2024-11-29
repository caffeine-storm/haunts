SHELL:=/bin/bash

TEST_REPORT_TAR:=testdata/report.tar.gz

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

clean:
	rm -f ${TEST_REPORT_TAR}

fmt:
	go fmt ./...

# -l for 'list files'
checkfmt:
	@gofmt -l ./

lint:
	go run github.com/mgechev/revive@v1.5.1 ./...

test:
	xvfb-run go test ${testrunargs}                     -tags nosound ./...

test-nocache:
	xvfb-run go test ${testrunargs} -count=1            -tags nosound ./...

test-dlv: singlepackage=${pkg}
test-dlv:
	dlv test --build-flags="-tags nosound" ${singlepackage} -- ${testrunargs}

devtest:
	xvfb-run go test ${testrunargs} -modfile dev.go.mod -tags nosound ./...

update-glop:
	go -C tools/update-glop/ run cmd/main.go
	go mod tidy

update-appveyor-image:
	go run tools/update-appveyor-image/main.go ./appveyor.yml

spawn-pprof:
	pprof -http :8080 ./perf.data

# Deliberately signal failure from this recipe so that CI notices failing tests
# are red.
appveyor-test-report-and-fail: test-report
	appveyor PushArtifact ${TEST_REPORT_TAR} -DeploymentName "test report tarball"
	false

test-report: ${TEST_REPORT_TAR}

${TEST_REPORT_TAR}:
	tar \
		--auto-compress \
		--create \
		--file $@ \
		--directory testdata/ \
		--files-from <(cd testdata; find  . -name '*.rej.*' | while read fname ; do \
				echo $$fname ; \
				echo $${fname/.rej} ; \
			done \
		)
# Let go tooling decide if things are out-of-date
.PHONY: haunts
.PHONY: devhaunts
.PHONY: clean
.PHONY: fmt lint
.PHONY: test devtest spawn-pprof
.PHONY: update-glop update-appveyor-image
