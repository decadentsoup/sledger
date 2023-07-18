#!/bin/sh -e

if echo "$0" | grep 'common\.sh' > /dev/null; then
	echo "common test routines -- do not run directly"
	exit 1
fi

step() {
	printf "\33[1m==> \33[35m$1\33[0m\n"
}

step "Change directory to the repository root."
cd "$(dirname "$0")/.."
