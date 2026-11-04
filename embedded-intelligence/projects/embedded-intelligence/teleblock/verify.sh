#!/bin/sh
# teleblock — project entry point for the repository arbiter.
#
# The arbiter lives at the repository root. `./verify.sh` is resolved against
# the shell's working directory, so from this directory it fails with a bare
# "/bin/sh: 1: ./verify.sh: not found". This shim walks up to the real script
# and hands it every argument.
#
#   ./verify.sh teleblock            # run the arbiter
#   ./verify.sh --print-arbiter      # show which script would run
#   VERIFY_SH=/path/to/verify.sh ./verify.sh
set -eu

PROJECT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

usage() {
	cat <<'EOF'
usage: ./verify.sh [--print-arbiter] [arbiter arguments...]

Run the repository arbiter for teleblock: the first verify.sh above this
project directory.

  --print-arbiter  print the arbiter path and exit, without running it
  -h, --help       show this message

Environment:
  VERIFY_SH        use this script instead of searching upward
EOF
}

# An explicit override, or the first verify.sh above the project directory.
# The project-local shim is deliberately out of the search path: resolving to
# itself would exec this file forever.
resolve_arbiter() {
	if [ -n "${VERIFY_SH:-}" ]; then
		printf '%s\n' "$VERIFY_SH"
		return 0
	fi
	dir=$PROJECT_DIR
	while [ "$dir" != "/" ] && [ "$dir" != "." ]; do
		dir=$(dirname -- "$dir")
		if [ -f "$dir/verify.sh" ] && [ "$dir/verify.sh" != "$PROJECT_DIR/verify.sh" ]; then
			printf '%s\n' "$dir/verify.sh"
			return 0
		fi
	done
	return 1
}

print_only=0
case "${1:-}" in
	-h|--help)       usage; exit 0 ;;
	--print-arbiter) print_only=1; shift ;;
esac

if ! arbiter=$(resolve_arbiter); then
	printf 'verify: no verify.sh above %s\n' "$PROJECT_DIR" >&2
	printf 'verify: set VERIFY_SH=/path/to/verify.sh, or run the arbiter from the repository root\n' >&2
	exit 3
fi

if [ ! -f "$arbiter" ]; then
	printf 'verify: %s does not exist\n' "$arbiter" >&2
	exit 3
fi
if [ ! -x "$arbiter" ]; then
	printf 'verify: %s is not executable — try: chmod +x %s\n' "$arbiter" "$arbiter" >&2
	exit 3
fi

if [ "$print_only" -eq 1 ]; then
	printf '%s\n' "$arbiter"
	exit 0
fi

exec "$arbiter" "$@"
