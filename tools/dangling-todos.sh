#!/bin/bash

set -eu

here=`dirname $0`
cd "$here/.."

# Find bug numbers that are defined in todo.txt
definedbugs=`sed -e 's,.*#\([0-9]*\).*,\1,' < todo.txt | grep '^[0-9]\+' | sort -n | uniq`

# Find bug numbers written in the code
referencedbugs=`ack 'TODO.*#[0-9]+' | sed 's,.*#\([0-9]*\).*,\1,' | sort -n | uniq`

# Assert that every bug reference in the code exists in todo.txt
for bug in $referencedbugs ; do
	grep $bug <(echo "$definedbugs") >/dev/null || echo "dangling bug: $bug"
done
