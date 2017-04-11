#!/usr/bin/env bash
set -o errexit -o nounset -o pipefail

if ((BASH_VERSINFO[0] < 4)); then
    echo >&2 "Bash version 4 required"
    exit 1
fi

script_path="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
prog_path="$script_path/../"

idx=0
failed=0

expect() {
    local expected="$1"
    local input="$3"

    idx="$((idx+1))"
    IFS= read -rd '' fixed   < <( echo -ne "$expected" ) || true
    IFS= read -rd '' result  < <( echo -ne "$input" | nlfix) || true
    if [[ "$fixed" != "$result" ]]; then
        echo -n "FAIL: line "
        caller
        echo "  expect $(printf %q "$fixed")"
        echo "  actual $(printf %q "$result")"
        failed="$((failed+1))"
    fi
}

nlfix() {
    "$prog_path/nlfix-test"
}

(cd "$prog_path"; make nlfix-test)

expect "\n\n" from "\r\n\r\n" 
expect "\n\n" from "\r\n\r" 
expect "\n\n" from "\n\r" 
expect "\n\n" from "\r\r\n" 
expect "\n\n\n\n" from "\r\r\r\r" 
expect "\n\n\n" from "\r\r\r\n" 
expect "\n\n\n\n" from "\n\n\n\r"

# these test handling at the boundaries of the reads
# this is tied to READ_SIZE in the makefile, which should
# be 16
expect "123456789012345\n123456789012345\n" \
    from "123456789012345\r\n123456789012345\r\n"

expect "1234567890123456\n1234567890123456\n" \
    from "1234567890123456\r\n1234567890123456\r\n"

expect "123456789012345\n\n123456789012345\n\n" \
    from "123456789012345\r\r123456789012345\r\r"

expect "1234567890123456\n1234567890123456\n" \
    from "1234567890123456\r1234567890123456\r\n"

echo "passed:$idx failed:$failed"
if [[ "$failed" -gt 0 ]]; then
    exit 2
fi

