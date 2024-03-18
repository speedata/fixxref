# Fix PDF Xref table

This little program fixes an XRef (cross reference) table in PDF files and updates `/Length` fields for streams. Warning: this is for educational purpose only and won't work on real world PDF files.

## Rationale

I sometimes create PDF files from scratch with a text editor. There is an easy part and a hard part. The easy part is to write objects like

```
3 0 obj
<<
    /Type /Page
    /MediaBox [ 0 0 200 200 ]
    /Contents 4 0 R
    /Parent 2 0 R
    /Resources << /Font << /F1 5 0 R  >>  >>
>>
endobj
```

which can be done with a text editor, such as VS Code.

The hard part is to find the exact byte positions of the object starts (`3 0 obj` in this case), collect the info and put this into a strange looking table, which looks like this (including the PDF trailer):

```
xref
0 6
0000000000 65535 f
0000000026 00000 n
0000000084 00000 n
0000000156 00000 n
0000000307 00000 n
0000000423 00000 n
trailer <<
/Root 1 0 R
/ID [<69DE6067104A4373E29071EECE32D4F9> <69DE6067104A4373E29071EECE32D4F9>]
/Size 6
>>
startxref
519
%%EOF
```

So to help with the hard part, fixxref looks through the file, counts the bytes to the beginning of each `^\d+ 0 obj` regexp and writes this info to the xref table. It removes an existing xref table.


## Limitations

`fixxref` has a lot of limitations.

* It works only with uncompressed PDF files
* It uses regular expressions, it does not understand what is in the PDF
* No support for xref streams

## Installation

    go install github.com/speedata/fixxref@latest


## Usage

    $ fixxref filename.pdf

Opens the PDF file, truncates it and create the new contents.

## License

MIT
