#!/bin/bash
#
# Called by "git commit" with no arguments.  The hook should
# exit with non-zero status after issuing an appropriate message if
# it wants to stop the commit.

output=`make checkfmt`

if [ $? -ne 0 ]; then
	cat 1>&2 <<EOF
'make checkfmt' failed
this usually indicates a compile failure

fix the failure then try again
(or pass '--no-verify' to 'git commit')
EOF

	exit 1
fi

if [ -n "$output" ]; then
	cat 1>&2 <<EOF
please run 'make fmt' first
(or pass '--no-verify' to 'git commit')

these files need formatting:
$output
EOF

	exit 1
fi
