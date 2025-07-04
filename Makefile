SHELL:=/bin/bash

TEST_REPORT_TAR:=test-report.tar.gz
PERF?=perf

ifneq "${testrun}" ""
testrunargs:=-run ${testrun}
else
testrunargs:=
endif

# On WSL, Xvfb thinks it can talk to hardware. Tell it not to.
XVFB_RUN:=LIBGL_ALWAYS_SOFTWARE=true xvfb-run -a --server-args="-screen 0 1920x1080x24"

SRC_DATADIR:=data

all: haunts

it: haunts
go: haunts ${SRC_DATADIR}
	./haunts

debug: devhaunts ${SRC_DATADIR}
	./$^

dev.go.mod: go.mod
	# dev.go.mod is just go.mod but patched to look at a local glop
	cat $^ | sed -e 's,\(runningwild/glop =>\).*,\1 ../deps-for-haunts/glop,' > $@

dev.go.sum: go.sum
	# dev.go.sum is just go.sum
	cp $^ $@

dlv: devhaunts ${SRC_DATADIR} dev.go.mod
	dlv debug --build-flags='-modfile dev.go.mod -tags nosound' .

devhaunts: dev.go.mod GEN_version.go
	go build -x -modfile dev.go.mod -o $@ -tags nosound main.go GEN_version.go

go.sum: go.mod
	go mod tidy

haunts: |go.mod go.sum
haunts: GEN_version.go
	go build -x -o $@ -tags nosound main.go $^

profile-haunts: haunts ${SRC_DATADIR}
	${PERF} record -g ./$^

profile-dev-haunts: devhaunts ${SRC_DATADIR}
	${PERF} record -g ./$^

# TODO(tmckee): this should use 'go gen' instead
GEN_version.go: tools/genversion/version.go .git/HEAD
	go run ./tools/genversion/cmd

clean:
	rm -f ${TEST_REPORT_TAR}
	rm -f devhaunts haunts
	find . \( \
		-name 'perf.data' \
		-or -name 'perf.data.old' \
		-or -name 'perftest' \
	\) -exec rm "{}" +

fmt:
	go fmt ./...
	git diff --no-color --name-status -- data/ | grep '^[MA]' | sed 's,^.\s\+,,' \
		| xargs -d '\n' go run ./tools/format-data-dir/

checkfmt:
	@# -l for 'list files'
	@gofmt -l ./
	@git diff --no-color --name-status -- data/ | grep '^[MA]' | sed 's,^.\s\+,,' \
		| xargs -d '\n' go run ./tools/format-data-dir/ --check

lint:
	go run github.com/mgechev/revive@v1.5.1 --config revive.toml ./...

test:
	${XVFB_RUN} go test ${testrunargs}                     -tags nosound ./...

test-verbose:
	${XVFB_RUN} go test ${testrunargs} -v                  -tags nosound ./...

test-racy:
	${XVFB_RUN} go test ${testrunargs} -count=1 -race      -tags nosound ./...

test-racy-with-cache:
	${XVFB_RUN} go test ${testrunargs}          -race      -tags nosound ./...

test-nocache:
	${XVFB_RUN} go test ${testrunargs} -count=1            -tags nosound ./...

test-fresh: |clean_rejects
test-fresh: test-nocache

pkg?= -- set 'pkg' to the package under test --
dlv-test: singlepackage=${pkg}
dlv-test: ${SRC_DATADIR}
# delve wants exactly one package at a time so "testrunargs" isn't what we
# want here. We use a var specifically for pointing at a single directory.
	[ -d "${singlepackage}" ] && \
	${XVFB_RUN} dlv test --build-flags="-tags nosound" ${singlepackage} -- ${testrunargs}

dlv-devtest: singlepackage=${pkg}
dlv-devtest: ${SRC_DATADIR}
# delve wants exactly one package at a time so "testrunargs" isn't what we
# want here. We use a var specifically for pointing at a single directory.
	[ -d "${singlepackage}" ] && \
	${XVFB_RUN} dlv test --build-flags="-modfile dev.go.mod -tags nosound" ${singlepackage} -- ${testrunargs}

.PRECIOUS: %/perftest
%/perftest: %
	go test -tags nosound -o ./$@ -c ./$^

.PRECIOUS: %/perf.data
%/perf.data: %/perftest
	cd $(dir $^) && \
	perf record -g -o perf.data ./perftest

devtest: dev.go.mod dev.go.sum
	${XVFB_RUN} go test ${testrunargs} -modfile dev.go.mod -tags nosound ./...

list_rejects:
	@find . -name testdata -type d | while read testdatadir ; do \
		find "$$testdatadir" -name '*.rej.*' ; \
	done

# opens expected and rejected files in 'feh'
view_rejects:
	@find . -name testdata -type d | while read testdatadir ; do \
		find "$$testdatadir" -name '*.rej.*' | while read rejfile ; do \
			echo -e >&2 "$${rejfile/.rej}\n$$rejfile" ; \
			echo "$${rejfile/.rej}" "$$rejfile" ; \
		done ; \
	done | xargs -r feh

clean_rejects:
	find . -name testdata -type d | while read testdatadir ; do \
		find "$$testdatadir" -name '*.rej.*' -exec rm "{}" + ; \
	done

promote_rejects:
	@find . -name testdata -type d | while read testdatadir ; do \
		find "$$testdatadir" -name '*.rej.*' | while read rejfile ; do \
			echo mv "$$rejfile" "$${rejfile/.rej}" ; \
			mv "$$rejfile" "$${rejfile/.rej}" ; \
		done \
	done

update-glop:
	go -C tools/update-glop/ run cmd/main.go
	go mod tidy

update-appveyor-image:
	go run tools/update-appveyor-image/main.go ./appveyor.yml

# TODO(tmckee): at least on WSL, getting errors that "Only 38% of samples had
# all locations mapped to a module, expected at least 95%". Presumably, this is
# to do with dynamic objects that have no source attribution. We ought to get a
# graphics driver stack that has symbols/hasn't been stripped.
spawn-pprof-%: %/perf.data
	pprof -http :8080 ./$^

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
		--files-from <(find  . -name '*.rej.*' | while read fname ; do \
				echo $$fname ; \
				echo $${fname/.rej} ; \
			done \
		)

trace-house-test:
	xvfb-run -a go test ${testrunargs} -exec ../tools/apitrace/trace-gl.sh -tags nosound ./house

# Let go tooling decide if things are out-of-date
.PHONY: haunts
.PHONY: devhaunts
.PHONY: clean fmt lint
.PHONY: devtest test test-fresh test-nocache test-report test-verbose
.PHONY: dlv-devtest dlv-test
.PHONY: clean_rejects list_rejects promote_rejects view_rejects
.PHONY: update-appveyor-image update-glop
