# recipes for running tests

include build/rejectfiles.mk

testrunpackages=./...
ifneq "${testrun}" ""
testrunargs:=-run ${testrun}
else
testrunargs:=
endif

ifeq "${pkg}" ""
testsinglepackageargs:=--set-'pkg'-to-a-package-dir
else
testsinglepackageargs="${pkg}"
# Replace each -$elem of 'testrunargs' with '-test.$elem'
newtestrunargs:=$(subst -,-test.,${testrunargs})
endif

# By default, the Xvfb instance will create a new Xauthority file in
# /tmp/xvfb-run.PID/Xauthority for access control.
# To interact with the Xvfb instance, you can set your XAUTHORITY and DISPLAY
# environment vars accordingly.
testing_with_xvfb=xvfb-run --server-args="-screen 0 512x64x24" --auto-servernum
testing_env=${testing_with_xvfb}

test:
	${testing_env} go test                   ${testrunargs} ${testrunpackages}

test-verbose:
	${testing_env} go test -v                ${testrunargs} ${testrunpackages}

test-racy:
	${testing_env} go test -count=1 -race    ${testrunargs} ${testrunpackages}

test-racy-with-cache:
	${testing_env} go test          -race    ${testrunargs} ${testrunpackages}

test-spec:
	${testing_env} go test -run ".*Specs"    ${testrunargs} ${testrunpackages}

test-nocache:
	${testing_env} go test -count=1          ${testrunargs} ${testrunpackages}

test-dlv:
# delve wants exactly one package at a time so "testrunpackages" isn't what we
# want here. We use a var specifically for pointing at a single directory.
	[ -d ${testsinglepackageargs} ] && \
	${testing_env} dlv test ${testsinglepackageargs} -- ${newtestrunargs}

test-fresh: |clean_rejects
test-fresh: test-nocache

