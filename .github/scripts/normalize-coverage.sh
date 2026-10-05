#!/usr/bin/env bash
set -euo pipefail

# Keep the library scope and merge blocks repeated by cross-package coverage.
# Usage: bash normalize-coverage.sh profile.out > normalized.out
awk '
NR == 1 { mode = $0; next }
$1 !~ /\/cmd\/|\/tests\// {
    statements[$1] = $2;
    hits[$1] += $3;
}
END {
    print mode;
    for (block in statements) print block, statements[block], hits[block];
}' "${1:?Coverage profile required}"
