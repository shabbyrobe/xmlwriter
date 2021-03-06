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
query=""
db=""
required=( xmllint make iconv )
for r in "${required[@]}"; do
    hash "$r" 2>/dev/null || { echo >&2 "Required program $r missing"; exit 1; }
done

while getopts ":f:l:d:q:" opt; do
    case "$opt" in
        l) filelist="$OPTARG";;
        f) file="$OPTARG";;
        d) db="$OPTARG";;
        q) query="$OPTARG";;
        *) echo >&2 "Invalid usage"; echo "$usage"; exit 1;
    esac
done

if [[ -n "$query" && -z "$db" ]]; then
    echo >&2 "Cannot use -q without -d"
    exit 1
fi

if [[ -z "$filelist" && -z "$file" && -z "$db" ]]; then
    echo >&2 "File list missing"
    exit 1
fi

echo "Building tools"
(cd gotester; go build )
(cd ctester ; make all )

input="$(mktemp)"
test_in="$(mktemp)"
ctest_out="$(mktemp)"
gotest_out="$(mktemp)"
orig_out="$(mktemp)"
error_file="$(mktemp)"

cleanup_files=("$input" "$test_in" "$error_file" "$ctest_out" "$gotest_out" "$orig_out")
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
        if [[ -n "$f" ]]; then
            rm -f "$f"
        fi
    done
}

xmlclean() {
    ./tools.sh xmlclean
}

skip() {
    skip="$((skip+1))"
    echo -e "\e[93mSKIP\e[0m\t$i\t$line\t$1"
}

fail() {
    fail="$((fail+1))"
    echo -e "\e[91mFAIL\e[0m\t$i\t$line\t$1"
}

trap cleanup INT TERM

i=0
pass=0
skip=0
fail=0

while read -r line; do
    if [[ ! -f "$line" ]]; then
        skip "file does not exist"
        continue
    fi

    mime="$( file -b --mime "$line" )"

    i="$((i+1))"
    # file --mime sees bare xml without a doc declaration as text/html
    if [[ "$mime" != "text/xml;"* && "$mime" != "application/xml;"* && "$mime" != "text/plain;"* && "$mime" != "text/html;"* ]]; then
        skip "mime type not text/xml, application/xml, text/html or text/plain; found '$mime'"
        continue
    fi
    enc="$( <"$line" encextractor || true )"
    if [[ -z "$enc" ]]; then
        if [[ "$mime" =~ charset=(.*) ]]; then
            # only saw these so far: 
            #   unknown-8bit us-ascii utf-16be utf-16le utf-8
            enc="${BASH_REMATCH[1]}"
        fi
    fi

    # I give up. it's not worth wasting any more time trying to work out
    # how to get expat to properly call the unknown encoding handler on
    # windows-1252, to work out why xmllint introduces extra whitespace
    # in certain situations when there is a CR, or to work out how to 
    # fix the normaliser so it doesn't destroy multibyte encodings that
    # aren't ascii-aware. out comes the big stick: everything goes to UTF-8, we
    # test encodings some other way. investing more time into this won't
    # find any more bugs.
    infile="$line"
    if [[ "${enc^^}" != "UTF-8" ]]; then 
        echo -e "\e[94mCONV\e[0m\t$enc\tUTF-8\t$line"
        if ! iconv -f "$enc" -t "UTF-8" "$line" | encfixer -f "UTF-8" > "$input"; then
            fail "iconv failed"
            continue
        fi
        infile="$input"
    fi

    if ! (( i % 100 )); then
        echo "$i done"
    fi

    rm -f "$error_file"
    rc=""

    # xmllint will spew errors to stderr and still return 0!
    <"$infile" xmlclean > "$orig_out" 2>"$error_file" || rc="$?"
    if [[ -s "$error_file" || -n "$rc" ]]; then
        skip "invalid xml"
        continue
    fi

    if [[ "$(filesize "$line")" -gt 10000000 ]]; then
        # the testbuilder files are HUGE by comparison to the input. a 1gb XML file
        # would be impossible to process on a machine with 8gb of ram without a pipe.
        if ! <"$orig_out" testbuilder | ctester | xmlclean > "$ctest_out"; then
            fail "ctester failed"
            continue
        fi
        if ! <"$orig_out" testbuilder | gotester | xmlclean > "$gotest_out"; then
            fail "gotester failed"
            continue
        fi
    else
        if ! <"$orig_out" testbuilder > "$test_in"; then
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

    if [[ -n "$rco" || -n "$rcc" ]]; then
        if [[ -n "$rco" ]]; then
            fail "orig does not compare to gowriter"
            diff -u <( xmllint --format "$orig_out" ) <( xmllint --format "$gotest_out" ) \
                >/tmp/result-"$i"-orig || true
        fi
        if [[ -n "$rcc" ]]; then
            fail "libxml does not compare to gowriter"
            diff -u <( xmllint --format "$ctest_out" ) <( xmllint --format "$gotest_out" ) \
                >/tmp/result-"$i"-ctest || true
        fi
        cp "$orig_out"   /tmp/out-"$i"-orig
        cp "$ctest_out"  /tmp/out-"$i"-ctest
        cp "$gotest_out" /tmp/out-"$i"-gotest
    else
        pass="$((pass+1))"
    fi

done < <(
    if [[ -n "$db" ]]; then
        indexer query "$db" "$query"
    fi
    if [[ -n "$file" ]]; then
        echo "$file"
    fi
    if [[ -n "$filelist" ]]; then
        cat "$filelist"
    fi
)

echo 
echo "done!"
echo -e "pass:\t$pass"
echo -e "fail:\t$fail"
echo -e "skip:\t$skip"

