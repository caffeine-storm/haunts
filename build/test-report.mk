# Recipes for generating and manipulating a 'test-report' tarball with
# rejection files inside. Useful for debuggin CI failures.
# Assumes that CI is running on ci.appveyor.com

TEST_REPORT_TAR:=test-report.tar.gz

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
				echo "$$fname" ; \
				echo "$${fname/.rej}" ; \
			done \
		)

test-report-clean:
	rm -f ${TEST_REPORT_TAR}

# Tell included makefiles that, if they've got a 'clean' recipe, it now depends
# on cleaning the test-report tarball too.
clean: test-report-clean

.PHONY: ${TEST_REPORT_TAR}
.PHONY: test-report test-report-clean
.PHONY: appveyor-test-report-and-fail
