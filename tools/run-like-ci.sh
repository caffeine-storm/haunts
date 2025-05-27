#!/bin/bash

# Run from the project root
here=`dirname $0`
cd "$here/.."

# --rm to remove container after exit
# --volume .:/haunts will mount the current haunts project on /haunts inside
#                    the container
# --interactive --tty for an interactive shell with a pseudo-TTY
podman run --rm --volume .:/haunts --interactive --tty docker.io/caffeinestorm/haunts-custom-build-image:latest bash
