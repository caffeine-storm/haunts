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
testing_with_xvfb=xvfb-run --server-args="-screen 0 1920x1080x24" --auto-servernum

ifeq "${testing_env}" ""
testing_env:=${testing_with_xvfb}
else
testing_env:=${testing_env} ${testing_with_xvfb}
endif

test:
	${testing_env} go test                 ${testbuildflags} ${testrunargs} ${testrunpackages}

test-verbose:
	${testing_env} go test -v              ${testbuildflags} ${testrunargs} ${testrunpackages}

test-racy:
	${testing_env} go test -count=1 -race  ${testbuildflags} ${testrunargs} ${testrunpackages}

test-racy-with-cache:
	${testing_env} go test          -race  ${testbuildflags} ${testrunargs} ${testrunpackages}

test-spec:
	${testing_env} go test -run ".*Specs"  ${testbuildflags} ${testrunargs} ${testrunpackages}

test-nocache:
	${testing_env} go test -count=1        ${testbuildflags} ${testrunargs} ${testrunpackages}

test-dlv:
# delve wants exactly one package at a time so "testrunpackages" isn't what we
# want here. We use a var specifically for pointing at a single directory.
	[ -d ${pkg} ] && \
	${testing_env} dlv test ${pkg} ${dlvbuildflags} -- ${newtestrunargs}

test-fresh: |clean_rejects
test-fresh: test-nocache
