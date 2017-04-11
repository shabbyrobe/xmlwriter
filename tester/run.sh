#!/usr/bin/env bash
set -o errexit -o nounset -o pipefail

if ((BASH_VERSINFO[0] < 4)); then
    echo >&2 "Bash version 4 required"
    exit 1
fi

script_path="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$script_path"

# we use a lot of tools that get built in these directories.
PATH="$script_path/ctester:$script_path/gotester:$PATH"

usage="Usage: run.sh ( -l <filelist> | -f <file> )"

filelist=""
file=""
required=( xmllint make )
for r in "${required[@]}"; do
    hash "$r" 2>/dev/null || { echo >&2 "Required program $r missing"; exit 1; }
done

while getopts ":f:l:" opt; do
    case "$opt" in
        l) filelist="$OPTARG";;
        f) file="$OPTARG";;
        *) echo >&2 "Invalid usage"; echo "$usage"; exit 1;
    esac
done

if [[ -z "$filelist" && -z "$file" ]]; then
    echo >&2 "File list missing"
    exit 1
fi

echo "Building tools"
(cd gotester; go build )
(cd ctester ; make all )

test_in="$(mktemp)"
ctest_out="$(mktemp)"
gotest_out="$(mktemp)"
orig_out="$(mktemp)"
error_file="$(mktemp)"
st1="$(mktemp)"
st2="$(mktemp)"
st3="$(mktemp)"

cleanup_files=("$st1" "$st2" "$st3" "$test_in" "$error_file" "$ctest_out" "$gotest_out" "$orig_out")
os="$(uname -s)"

filesize() {
    if [[ "$os" == "Darwin" ]]; then
        stat -f%z "$1"
    else
        stat -c%s "$1"
    fi
}

xmllint() {
    /usr/bin/env xmllint --nonet "$@"
}

cleanup() {
    local f
    jobs -p | xargs kill >/dev/null 2>&1
    for f in "${cleanup_files[@]}"; do
        if [[ ! -z "$f" ]]; then
            rm -f "$f"
        fi
    done
}

xmlclean() {
    # this is crazy. a structural xml differ might be better, but this actually
    # works, believe it or not. the xml differs i tried were slow and didn't work
    # at all.
    attrsorter | encfixer | xmllint --format -
}

skip() {
    skip="$((skip+1))"
    echo -e "SKIP\t$i\t$line: $1"
}

fail() {
    fail="$((fail+1))"
    echo -e "FAIL\t$i\t$line: $1"
}

trap cleanup INT TERM

i=0
pass=0
skip=0
fail=0

while read line; do
    mime="$( file -b --mime "$line" )"

    i="$((i+1))"
    # file --mime sees bare xml without a doc declaration as text/html
    if [[ "$mime" != "application/xml;"* && "$mime" != "text/plain;"* && "$mime" != "text/html;"* ]]; then
        skip "mime type not application/xml, text/html or text/plain"
        continue
    fi
    enc="$( <"$line" encextractor || true )"
    if [[ ! -z "$enc" ]]; then
        case "$enc" in
            UTF-8) ;;
            ISO-8859-1) ;;
            *) skip "mime $enc not supported"; continue;;
        esac
    else
        if [[ "$mime" != *"; charset=utf-8" ]]; then
            if ! <"$line" nohigh; then
                skip "file is not utf-8 or contains ASCII high bytes"
                continue
            fi
        fi
    fi

    if ! (( i % 100 )); then
        echo "$i done"
    fi

    rm -f "$error_file"
    rc=""
    <"$line" nlfix | xmlclean > "$orig_out" 2>"$error_file" || rc="$?"
    if [[ -s "$error_file" || ! -z "$rc" ]]; then
        skip "invalid xml"
        continue
    fi

    # it would be better to use no temp files at all if we could -
    # tee can split the output off to multiple processes but we can't
    # easily capture the exit status.

    if [[ "$(filesize "$line")" -gt 10000000 ]]; then
        # the testbuilder files are HUGE by comparison to the input. a 1gb XML file
        # would be impossible to process on a machine with 8gb of ram without a pipe.
        if ! <"$line" nlfix | testbuilder | ctester | xmlclean > "$ctest_out"; then
            fail "ctester failed"
            continue
        fi
        if ! <"$line" nlfix | testbuilder | gotester | xmlclean > "$gotest_out"; then
            fail "gotester failed"
            continue
        fi
    else
        if ! <"$line" nlfix | testbuilder > "$test_in"; then
            fail "testbuilder failed"
            continue
        fi
        if ! <"$test_in" ctester | xmlclean > "$ctest_out"; then
            fail "ctester failed"
            continue
        fi
        if ! <"$test_in" gotester | xmlclean > "$gotest_out"; then
            fail "gotester failed"
            continue
        fi
    fi

    rco=""
    rcc=""
    cmp -s "$orig_out"  "$gotest_out" || rco=$?
    cmp -s "$ctest_out" "$gotest_out" || rcc=$?

    if [[ ! -z "$rco" || ! -z "$rcc" ]]; then
        if [[ ! -z "$rco" ]]; then
            fail "orig does not compare to gowriter"
            diff -u "$orig_out" "$gotest_out" > /tmp/result-"$i"-orig || true
        fi
        if [[ ! -z "$rcc" ]]; then
            fail "libxml does not compare to gowriter"
            diff -u "$ctest_out" "$gotest_out" > /tmp/result-"$i"-ctest || true
        fi
        cp "$orig_out"   /tmp/out-"$i"-orig
        cp "$ctest_out"  /tmp/out-"$i"-ctest
        cp "$gotest_out" /tmp/out-"$i"-gotest
    else
        pass="$((pass+1))"
    fi

done < <(
    if [[ ! -z "$file" ]]; then
        echo "$file"
    fi
    if [[ ! -z "$filelist" ]]; then
        cat "$filelist"
    fi
)

echo 
echo "done!"
echo -e "pass:\t$pass"
echo -e "fail:\t$fail"
echo -e "skip:\t$skip"

