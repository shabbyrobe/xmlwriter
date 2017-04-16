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

if [[ ! -z "$query" && -z "$db" ]]; then
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
        if [[ ! -z "$f" ]]; then
            rm -f "$f"
        fi
    done
}

xmlclean() {
    # this is crazy. a structural xml differ might be better, but this actually
    # works, believe it or not. the xml differs i tried were slow and didn't work
    # at all.
    #
    # `normaliser`
    #     Attributes can get written in any order, but elements with the same
    #     attributes are semantically identical regardless of the order. This
    #     utility re-parses the xml and sorts the attributes so they can be
    #     compared.
    # 
    # `encfixer`
    #     `libxml` won't allow an arbitrarily cased encoding, so if the input
    #     file supplies one, for e.g. 'utf-8' instead of 'UTF-8', there's no
    #     way to get the C writer to emit the lowercase version. This re-
    #     parses the encoding and uppercases it, then thumps the rest of the file
    #     to stdout unmolested.
    #
    # -noblanks is a compromise - expat doesn't play nicely with CR characters.
    # see /opt/local/share/djvu/osi/de/libdjvu++.xml
    # and /Applications/Unity/Unity.app/Contents/UnityExtensions/Unity/EditorTestsRunner/Editor/nunit.framework.xml
    # dos2unix and my own alternative non-solution 'nlfix' both just move
    # the problem around a bit, they don't elminate it. the "solution" appears
    # to be to strip unimportant blanks from the source using xmllint.
    normaliser | encfixer | xmllint --noblanks -
}

skip() {
    skip="$((skip+1))"
    echo -e "SKIP\t$i\t$line\t$1"
}

fail() {
    fail="$((fail+1))"
    echo -e "FAIL\t$i\t$line\t$1"
}

trap cleanup INT TERM

i=0
pass=0
skip=0
fail=0

while read line; do
    if [[ ! -f "$line" ]]; then
        skip "file does not exist"
        continue
    fi

    mime="$( file -b --mime "$line" )"

    i="$((i+1))"
    # file --mime sees bare xml without a doc declaration as text/html
    if [[ "$mime" != "application/xml;"* && "$mime" != "text/plain;"* && "$mime" != "text/html;"* ]]; then
        skip "mime type not application/xml, text/html or text/plain"
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
        echo -e "CONV\t$enc\tUTF-8\t$line"
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
    if [[ -s "$error_file" || ! -z "$rc" ]]; then
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

    if [[ ! -z "$rco" || ! -z "$rcc" ]]; then
        if [[ ! -z "$rco" ]]; then
            fail "orig does not compare to gowriter"
            diff -u <( xmllint --format "$orig_out" ) <( xmllint --format "$gotest_out" ) \
                >/tmp/result-"$i"-orig || true
        fi
        if [[ ! -z "$rcc" ]]; then
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
    if [[ ! -z "$db" ]]; then
        indexer query "$db" "$query"
    fi
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

