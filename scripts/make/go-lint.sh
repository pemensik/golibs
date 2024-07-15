#!/bin/sh

# This comment is used to simplify checking local copies of the script.  Bump
# this number every time a significant change is made to this script.
#
# AdGuard-Project-Version: 8

verbose="${VERBOSE:-0}"
readonly verbose

if [ "$verbose" -gt '0' ]
then
	set -x
fi

# Set $EXIT_ON_ERROR to zero to see all errors.
if [ "${EXIT_ON_ERROR:-1}" -eq '0' ]
then
	set +e
else
	set -e
fi

set -f -u



# Source the common helpers, including not_found and run_linter.
. ./scripts/make/helper.sh



# Simple analyzers

# blocklist_imports is a simple check against unwanted packages.  The following
# packages are banned:
#
#   *  Package errors is replaced by our own package in the
#      github.com/AdguardTeam/golibs module.
#
#   *  Packages golang.org/x/exp/slices and golang.org/x/net/context have been
#      moved into stdlib.
#
#   *  Packages log and github.com/AdguardTeam/golibs/log are replaced by
#      stdlib's new package log/slog and AdGuard's new utilities package
#      github.com/AdguardTeam/golibs/logutil/slogutil.
#
#   *  Package io/ioutil is soft-deprecated.
#
#   *  Package reflect is often an overkill, and for deep comparisons there are
#      much better functions in module github.com/google/go-cmp.  Which is
#      already our indirect dependency and which may or may not enter the stdlib
#      at some point.
#
#      See https://github.com/golang/go/issues/45200.
#
#   *  Package sort is replaced by package slices.
#
#   *  Package unsafe is… unsafe.
#
# Currently, the only standard exception are files generated from protobuf
# schemas, which use package reflect.  If your project needs more exceptions,
# add and document them.
#
# TODO(a.garipov): Add deprecated package golang.org/x/exp/maps once all
# projects switch to Go 1.23.
blocklist_imports() {
	# TODO(e.burkov):  These are temporary exclusions for some packages.
	git grep\
		-e '[[:space:]]"unsafe"$'\
		-n\
		-- '*.go'\
		':!cache/data.go'\
		':!cache/list.go'\
		| sed -e 's/^\([^[:space:]]\+\)\(.*\)$/\1 blocked import:\2/'\
		|| exit 0

	git grep\
		-e '[[:space:]]"errors"$'\
		-n\
		-- '*.go'\
		':!errors/errors.go'\
		| sed -e 's/^\([^[:space:]]\+\)\(.*\)$/\1 blocked import:\2/'\
		|| exit 0

	git grep\
		-e '[[:space:]]"reflect"$'\
		-n\
		-- '*.go'\
		':!testutil/testutil.go'\
		':!netutil/urlutil/url.go'\
		| sed -e 's/^\([^[:space:]]\+\)\(.*\)$/\1 blocked import:\2/'\
		|| exit 0

	git grep\
		-e '[[:space:]]"log"$'\
		-n\
		-- '*.go'\
		':!log/log.go'\
		':!logutil/slogutil/slogutil.go'\
		| sed -e 's/^\([^[:space:]]\+\)\(.*\)$/\1 blocked import:\2/'\
		|| exit 0

	# These are common for the whole code.
	git grep\
		-e '[[:space:]]"io/ioutil"$'\
		-e '[[:space:]]"sort"$'\
		-e '[[:space:]]"golang.org/x/exp/slices"$'\
		-e '[[:space:]]"golang.org/x/net/context"$'\
		-n\
		-- '*.go'\
		':!*.pb.go'\
		| sed -e 's/^\([^[:space:]]\+\)\(.*\)$/\1 blocked import:\2/'\
		|| exit 0
}

# method_const is a simple check against the usage of some raw strings and
# numbers where one should use named constants.
method_const() {
	git grep -F\
		-e '"DELETE"'\
		-e '"GET"'\
		-e '"PATCH"'\
		-e '"POST"'\
		-e '"PUT"'\
		-n\
		-- '*.go'\
		| sed -e 's/^\([^[:space:]]\+\)\(.*\)$/\1 http method literal:\2/'\
		|| exit 0
}

# underscores is a simple check against Go filenames with underscores.  Add new
# build tags and OS as you go.  The main goal of this check is to discourage the
# use of filenames like client_manager.go.
underscores() {
	underscore_files="$(
		git ls-files '*_*.go'\
			| grep -F\
			-e '_darwin.go'\
			-e '_generate.go'\
			-e '_linux.go'\
			-e '_test.go'\
			-e '_unix.go'\
			-e '_windows.go'\
			-e '_others.go'\
			-v\
			| sed -e 's/./\t\0/'
	)"
	readonly underscore_files

	if [ "$underscore_files" != '' ]
	then
		echo 'found file names with underscores:'
		echo "$underscore_files"
	fi
}

# TODO(a.garipov): Add an analyzer to look for `fallthrough`, `goto`, and `new`?



# Checks

run_linter -e blocklist_imports

run_linter -e method_const

run_linter -e underscores

run_linter -e gofumpt --extra -e -l .

# TODO(a.garipov): golint is deprecated, find a suitable replacement.

run_linter "${GO:-go}" vet ./...

run_linter govulncheck ./...

run_linter gocyclo --over 10 .

run_linter gocognit --over 10 .

run_linter ineffassign ./...

run_linter unparam ./...

git ls-files -- 'Makefile' '*.conf' '*.go' '*.mod' '*.sh' '*.yaml' '*.yml'\
	| xargs misspell --error\
	| sed -e 's/^/misspell: /'

run_linter nilness ./...

# TODO(e.burkov):  Add cache package.
run_linter fieldalignment\
	./container/\
	./errors/\
	./hostsfile/\
	./httphdr/\
	./internal/...\
	./ioutil/\
	./log/\
	./logutil/...\
	./mapsutil/\
	./mathutil/\
	./netutil/...\
	./osutil/\
	./service/\
	./stringutil/\
	./syncutil/\
	./testutil/...\
	./timeutil/\
	;

run_linter -e shadow --strict ./...

# TODO(e.burkov):  Add cache package.
run_linter gosec --quiet\
	./container/\
	./errors/\
	./hostsfile/\
	./httphdr/\
	./internal/...\
	./ioutil/\
	./log/\
	./logutil/...\
	./mapsutil/\
	./mathutil/\
	./netutil/...\
	./osutil/\
	./service/\
	./stringutil/\
	./syncutil/\
	./testutil/...\
	./timeutil/\
	;

run_linter errcheck ./...

staticcheck_matrix='
darwin:  GOOS=darwin
linux:   GOOS=linux
freebsd: GOOS=freebsd
openbsd: GOOS=openbsd
windows: GOOS=windows
'
readonly staticcheck_matrix

echo "$staticcheck_matrix" | run_linter staticcheck --matrix ./...
