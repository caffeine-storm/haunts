all: haunts

SHELL:=/bin/bash

# Let go tooling decide if things are out-of-date
.PHONY: haunts devhaunts
.PHONY: it clean fmt lint
.PHONY: devtest
.PHONY: update-appveyor-image update-glop

DATADIR:=data
PERF?=perf

# On WSL, Xvfb thinks it can talk to hardware. Tell it not to.
testing_env:=LIBGL_ALWAYS_SOFTWARE=true

it: haunts
go: haunts ${DATADIR}
	./haunts

lvl1: haunts ${DATADIR}
	./haunts lvl1

GENERATED_TARGETS=game/side_string.go cmd/gen/version.go
cmd/gen/version.go: .git/HEAD
	go generate ./cmd/

game/side_string.go:
	go generate ./game/

debug: devhaunts ${DATADIR}
	./$^

GLOPLOC?=../glop
dev.go.mod: go.mod
	# dev.go.mod is just go.mod but patched to look at a local glop
	cp $^ $@
	echo "replace github.com/caffeine-storm/glop => ${GLOPLOC}" >> $@

dev.go.sum: go.sum
	# dev.go.sum is just go.sum
	cp $^ $@

dlv: devhaunts ${DATADIR} dev.go.mod
	dlv debug --build-flags='-modfile dev.go.mod -tags nosound' .

devhaunts: dev.go.mod ${GENERATED_TARGETS}
	go build -x -modfile dev.go.mod -o $@ -tags nosound main.go

go.sum: go.mod
	go mod tidy

haunts: go.mod go.sum ${GENERATED_TARGETS}
haunts: main.go
	go build -x -o $@ -tags nosound main.go

profile-haunts: haunts ${DATADIR}
	${PERF} record --call-graph fp -- ./$^

profile-dev-haunts: devhaunts ${DATADIR}
	${PERF} record --call-graph fp -- ./$^

clean:
	rm -f devhaunts haunts
	find . \( \
		-name 'perf.data' \
		-or -name 'perf.data.old' \
		-or -name 'perftest' \
	\) -exec rm "{}" +

include build/gofumpt.mk
fmt: gofmt
	git diff --no-color --name-status -- data/ | grep '^[MA]' | sed 's,^.\s\+,,' \
		| xargs -d '\n' go run ./tools/format-data-dir/

checkfmt: checkgofmt
	@git diff --no-color --name-status -- data/ | grep '^[MA]' | sed 's,^.\s\+,,' \
		| xargs -d '\n' go run ./tools/format-data-dir/ --check

lint:
	go run github.com/mgechev/revive@v1.5.1 --config revive.toml ./...

testrunargs:=
testdeps:=${GENERATED_TARGETS}
testbuildflags:=-tags nosound
dlvbuildflags:=--build-flags="${testbuildflags}"
include build/testing-env.mk

.PRECIOUS: %/perftest
%/perftest: %
	go test -tags nosound -o ./$@ -c ./$^

.PRECIOUS: %/perf.data
%/perf.data: %/perftest
	cd $(dir $^) && \
	${PERF} record --callgraph fp -o perf.data ./perftest

test-dev: dev.go.mod dev.go.sum
	${testing_env} go test ${testrunargs} -modfile dev.go.mod -tags nosound ./...

test-dlvdev: dev.go.mod dev.go.sum
# delve wants exactly one package at a time so "testrunpackages" isn't what we
# want here. We use a var specifically for pointing at a single directory.
	[ -d ${pkg} ] && \
	${testing_env} dlv test ${pkg} --build-flags="${testbuildflags} -modfile dev.go.mod" -- ${newtestrunargs}

.PHONY: test-dev test-dlvdev

include build/test-report.mk

update-glop:
	go -C tools/update-glop/ run cmd/main.go
	go mod tidy
	
update-buildlib:
	go run ./tools/update-buildlib/cmd/main.go

update-appveyor-image:
	go run tools/update-appveyor-image/main.go ./appveyor.yml

# TODO(tmckee): at least on WSL, getting errors that "Only 38% of samples had
# all locations mapped to a module, expected at least 95%". Presumably, this is
# to do with dynamic objects that have no source attribution. We ought to get a
# graphics driver stack that has symbols/hasn't been stripped.
spawn-pprof-%: %/perf.data
	pprof -http :8080 ./$^

trace-house-test:
	${testing_env} go test ${testrunargs} -exec ../tools/apitrace/trace-gl.sh -tags nosound ./house
