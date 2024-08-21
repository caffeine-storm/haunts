#!/bin/bash

# This is a catch-all test that will check any new dependency by default.

set -e

rootdir=`dirname $0`/../..
cd $rootdir

# TODO(tmckee): verify non-go code too

# TODO(tmckee): mrg/cmwc is 'all-rights-reserved'; need to cut dependency
# Just a serialisable PRNG.

# TODO(tmckee): mrg/fmod is 'a-r-r'; need to find an alternative or roll our
# own fmod bindings

# TODO(tmckee): mrg/fsnotify is 'a-r-r'; the upstream repo _is_ licenced but
# not maintained. Look at github.com/fsnotify/fsnotify

# TODO(tmckee): mrg/opengl is 'a-r-r'; try to switch to
# https://github.com/go-gl-legacy/gl

# TODO(tmckee): rw/yedparse is 'a-r-r'; need to cut dependency (loads yFiles' xml dumps of graphs) Note: this is inside of glop

# Walk our dependency tree and assert that everything has a compatible licence.
# Note use a threshold of 0.85 to accept the BSD-3-Clause in glop (even though
# it's reported as BSD-4-Clause 🙃.

# Note: some manual verification was done for freetype; it's got a valid
# licence, just not one that go-licenses understands.
go-licenses check --ignore code.google.com/p/freetype-go --confidence_threshold 0.85 .
