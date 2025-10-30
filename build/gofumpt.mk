SHELL:=/bin/bash

# So... we want to run $2 as a pipeline stage downstream of $1. BUT! we want to
# be running against stderr, not stdin.
#
# The way to do this is "stash" stdout in a temporary file descriptor, 3.
# Redirect stderr to stdout, pipe that through the filter, then un-redirect
# streams to their original state.
#
# Bonus: we use the `pipefail` option in bash to make sure errors from $1 don't
# go missing.
define ErrPipe
	( set -o pipefail ; $1 2>&1 1>&3 | $2 1>&2 ) 3>&1
endef

# We want to grep out a pattern of error messages from stderr. Use ErrPipe for
# 'selecting' stderr, "grep -v" for exclusion but always return "success" from
# the filter stage to prevent spurious recipe failures.
define FilterError
	@$(call ErrPipe,$1,{ grep -v $2; true; })
endef

define SuppressGoVersionWarningNoEcho
	$(call FilterError,$1,"requires go .* switching to go")
endef

define SuppressGoVersionWarning
	@echo $1
	$(call SuppressGoVersionWarningNoEcho,$1)
endef

# Use gofumpt to enforce a consistent, opinionated style for go code.
gofmt:
	$(call SuppressGoVersionWarning,go run mvdan.cc/gofumpt@v0.9.2 -l -w .)

# Use checkgofmt to _check_ if there are style violations. Useful for
# pre-commit hooks.
checkgofmt:
	$(call SuppressGoVersionWarningNoEcho,go run mvdan.cc/gofumpt@v0.9.2 -l .)

.PHONY: gofmt checkgofmt
