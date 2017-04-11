Go xmlwriter Tester
===================

This tester is made up of a few key pieces - `gotester`, `testbuilder` and `ctester`,
and an XML-based testing DSL.


Test spec
---------

The core of the tester is an XML specification for sending commands to an
XMLWriter. A typical test looks like this:

```xml
<?xml version="1.0"?>
<script name="mytest">
    <command action="start" kind="elem" name="pants"/>
    <command action="write" kind="attr" name="attr">value</command>
    <command action="write" kind="attr" name="attr">value</command>
    <command action="end"   kind="elem" />
</script>
```

Tests can be piped to `gotester/gotester`, which will interpret the commands as 
as series of calls to an `xmlwriter.Writer{}`. The XML produced by that writer
will be written to stdout.

The `action` attribute can be either `start`, `write` or `end`. Node kinds must
implement `Startable` to be passed to `start`, and `Writable` to be passed to
`write`. The `end` API doesn't exactly match the golang library's API to
provide compatibility with the crazy `ctester` program, which is described
later.

See the documentation for the
[NodeKind](http://godoc.org/github.com/shabbyrobe/xmlwriter/#NodeKind) enum in
the xmlwriter library for the list of node kinds that can be used in the `kind`
attribute. To convert from the constant name, drop the `-Node` suffix and
convert from `CamelCase` to `hyphen-separated-lowercase`, for e.g.
`DTDAttListNode` becomes `dtd-att-list`.


Test runners: gotester and ctester
----------------------------------

`gotester` takes tests created using the above spec and runs them using
`xmlwriter.Writer{}`. It can be found in the `gotester/` directory.

`ctester` does the same thing, but using libxml2. It was created in order to
ensure that the go `xmlwriter.Writer{}` was producing semantically identical
output to a reference implementation. It can be found in the `ctester/`
directory. To build:

    ( cd ctester; make ctester )


Test builder: now things get really crazy
-----------------------------------------

Ok so now we have two testers - one written in go for the xmlwriter API, one
written in C for the libxml2 API. What if we could take ANY XML file at all,
run it through an XML reader (in this case expat because none of libxml2's
various readers did the job properly), and emit `<command>` elements that would
instruct either of our two test binaries to rebuild the same XML as we started with?

Enter the `testbuilder`. That's what that does. Pipe any XML file at all to it
and it will create a test according the above spec:

    # make a test case from an XML file
    cat ctester/test/testbuilder/dtd-04-in.xml | ctester/testbuilder

You can then run that test through both `ctester` and `gotester`, and diff the
results:

    # oh my god it's the same (pretty much):
    cat ctester/test/testbuilder/dtd-04-in.xml | ctester/testbuilder | gotester/gotester
    cat ctester/test/testbuilder/dtd-04-in.xml | ctester/testbuilder | ctester/ctester

OR you can use `./run.sh` and have it do all that for you:

    ./run.sh -f /path/to/your.xml

Ok, now I need to find some XML files to test with. Hang on...

    $ find / -type f -name '*.xml' | wc -l
       26564

Thank you, that will do nicely.

    ./run.sh -l /path/to/file/containing/list/of/xml/files.txt
    ./run.sh -l <( find /path/to/tree/containing/xmls -type f -name '*.xml' )

To build:

    sudo apt install libxml2
    ( cd ctester; make )

This process is a bit fraught and requires a few additional terrible C programs,
also found in the `ctester` folder along with the venerable `diff` utility.
Keep in mind that all of this crap is quite weakly tested, but it's a damn
sight faster and more accurate than the xml differs I tried.

`nlfix`
    `xmllint` doesn't coalesce non-significant whitespace the same way if
    the newlines aren't normalised first. `expat` (used in testbuilder)
    always replaces CR with LF (see lib/xmlparse.c:2629 or thereabouts),
    so `testbuilder` won't emit CR even if it is present in the original
    document. Y U NO dos2unix? Slower, more complicated, less predictable.

`attrsorter`
    Attributes can get written in any order, but elements with the same
    attributes are semantically identical regardless of the order. This
    utility re-parses the xml and sorts the attributes so they can be
    compared.

`encfixer`
    `libxml` won't allow an arbitrarily cased encoding, so if the input
    file supplies one, for e.g. 'utf-8' instead of 'UTF-8', there's no
    way to get the C writer to emit the lowercase version. This re-
    parses the encoding and uppercases it, then thumps the rest of the file
    to stdout unmolested.

