Go xmlwriter Tester
===================

This tester is made up of a few key pieces - `gotester`, `testbuilder` and
`ctester`, and an XML-based testing DSL

Note: this will eventually be moved into a separate repository when it
stabilises so that users of tools like `dep` or `govendor` aren't forced to
vendor the `charmap` extension when including `xmlwriter`.


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

Enter the `testbuilder`. That's what that does. Pipe any XML file -- preferably
UTF-8 encoded -- to it and it will create a test according the above spec:

    # make a test case from an XML file
    cat ctester/test/testbuilder/dtd-04-in.xml | ctester/testbuilder

You can then run that test through both `ctester` and `gotester`, and diff the
results:

    # oh my god it's the same (pretty much):
    cat ctester/test/testbuilder/dtd-04-in.xml | ctester/testbuilder | gotester/gotester
    cat ctester/test/testbuilder/dtd-04-in.xml | ctester/testbuilder | ctester/ctester

Or you can use `./run.sh` and have it do all that for you:

    ./run.sh -f /path/to/your.xml

Ok, now I need to find some XML files to test with. Hang on...

    $ find / -type f -name '*.xml' | wc -l
       26564

Thank you, that will do nicely.

    ./run.sh -l /path/to/file/containing/list/of/xml/files.txt
    ./run.sh -l <( find /path/to/tree/containing/xmls -type f -name '*.xml' )

You can also index them into a table and use that to drive the list. This makes
it easier to find files to test with that exhibit certain characteristics that
you might want to test:

    $ find / -type f -name '*.xml' | ctester/indexer index /path/to/db.sqlite
    $ ctester/indexer query 'elems > 0 OR nselems > 0'
    $ ./run.sh -d /path/to/db.sqlite -q 'elems > 0 OR nselems > 0'

To build:

    sudo apt install libxml2 iconv libsqlite3-dev libexpat-dev libmagic-dev
    ( cd ctester; make )

This process is a bit fraught and requires a few additional terrible C programs,
also found in the `ctester` folder along with the venerable `diff` utility.
Keep in mind that all of this crap is quite weakly tested, but it's a damn
sight faster and more accurate than the xml differs I tried.


Known Issues
------------

If your input XML contains the entities `&#xD;` or `&#13;`, they will cause the
output diff to fail.

Doctype entities will appear different if the input contains entities - expat
expands them and no part of our terrible pipeline of normalisation makes it
easy to make them diffable.

Different encodings are run through iconv to produce UTF-8 before the testbuilder
is run.

Input must be parseable by expat - many files on my hard drive are not.

Input must be a well-formed XML document: this means it must have an xml declaration
at the top and one and only one root element. Many files with the .xml
extension on my hard drive do not satisfy this.

`ctester.c` does not work with entities that contain double quotes in the
content. They are written by libxml's `xmlTextWriterWriteDTDEntity` with double
quotes regardless. Should follow this up with libxml, see if it's a bug at
their end.

