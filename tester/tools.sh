#!/bin/bash

script_path="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

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
    "$script_path"/ctester/normaliser | "$script_path"/ctester/encfixer | xmllint --noblanks -
}

"$1" "${@:-2}"

