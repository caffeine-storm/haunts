# recipes for running tests

include build/rejectfiles.mk

.PHONY: test \
	test-verbose \
	test-racy \
	test-racy-with-cache \
	test-spec \
	test-nocache \
	test-dlv \
	test-fresh

testdeps?=

testrunpackages:=./...
ifneq "${testrun}" ""
testrunargs:=${testrunargs} -run ${testrun}
endif

pkg?=--set-'pkg'-to-a-package-dir
dlvbuildflags?=
# Replace each -$elem of 'testrunargs' with '-test.$elem'
newtestrunargs:=$(subst -,-test.,${testrunargs})

# By default, the Xvfb instance will create a new Xauthority file in
# /tmp/xvfb-run.PID/Xauthority for access control.
# To interact with the Xvfb instance, you can set your XAUTHORITY and DISPLAY
# environment vars accordingly.
testing_with_xvfb=xvfb-run --server-args="-screen 0 1024x750x24" --auto-servernum

ifeq "${testing_env}" ""
testing_env:=${testing_with_xvfb}
else
testing_env:=${testing_env} ${testing_with_xvfb}
endif

test: ${testdeps}
	${testing_env} go test                 ${testbuildflags} ${testrunargs} ${testrunpackages}

test-verbose: ${testdeps}
	${testing_env} go test -v              ${testbuildflags} ${testrunargs} ${testrunpackages}

test-racy: ${testdeps}
	${testing_env} go test -count=1 -race  ${testbuildflags} ${testrunargs} ${testrunpackages}

test-racy-with-cache: ${testdeps}
	${testing_env} go test          -race  ${testbuildflags} ${testrunargs} ${testrunpackages}

test-spec: ${testdeps}
	${testing_env} go test -run ".*Specs"  ${testbuildflags} ${testrunargs} ${testrunpackages}

test-nocache: ${testdeps}
	${testing_env} go test -count=1        ${testbuildflags} ${testrunargs} ${testrunpackages}

test-dlv: ${testdeps}
# delve wants exactly one package at a time so "testrunpackages" isn't what we
# want here. We use a var specifically for pointing at a single directory.
	[ -d ${pkg} ] && \
	${testing_env} dlv test ${pkg} ${dlvbuildflags} -- ${newtestrunargs}

test-fresh: |clean_rejects
test-fresh: test-nocache
