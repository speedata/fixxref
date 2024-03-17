package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	startObj = regexp.MustCompile(`^(\d+) 0 obj$`)
)

type onum int

func writePDFFile(filename string, contents string) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	if err = f.Truncate(0); err != nil {
		return err
	}
	if _, err = f.WriteString(contents); err != nil {
		return err
	}
	return f.Close()
}

// scan reads the lines until the xref is found or until the end, whichever comes first. Objects with a comment
//
//	%% Info or %% Root
//
// right before the object number will be treated as info or root objects in the trailer.
//
// The return string is a complete PDF file which replaces the input.
func scan(r io.Reader) (string, error) {
	var ret strings.Builder

	sc := bufio.NewScanner(r)
	sc.Split(bufio.ScanLines)
	pos := 0
	var nextObjIsInfo, nextObjIsRoot bool
	var infoObject, rootObject, maxonum onum
	objectPositions := make(map[onum]int)
	objectPositions[0] = 0
	var xrefpos int
	for sc.Scan() {
		line := sc.Text()
		if res := startObj.FindAllStringSubmatch(line, -1); len(res) > 0 {
			objnum, err := strconv.Atoi(res[0][1])
			if err != nil {
				return "", err
			}
			if nextObjIsInfo {
				infoObject = onum(objnum)
				nextObjIsInfo = false
			} else if nextObjIsRoot {
				rootObject = onum(objnum)
				nextObjIsRoot = false
			}
			objectPositions[onum(objnum)] = pos
			maxonum = onum(objnum) + 1
			fmt.Fprintln(&ret, line)
		} else if strings.HasPrefix(line, "xref") {
			// found startxref, remove everything from here
			xrefpos = pos
			goto writeXRef
		} else if strings.HasPrefix(line, "%% Info") {
			nextObjIsInfo = true
			fmt.Fprintln(&ret, line)
		} else if strings.HasPrefix(line, "%% Root") {
			nextObjIsRoot = true
			fmt.Fprintln(&ret, line)
		} else {
			fmt.Fprintln(&ret, line)
		}
		pos += len(line) + 1
	}

	xrefpos = pos
writeXRef:
	type chunk struct {
		startOnum onum
		positions []int64
	}
	objectChunks := []chunk{}
	var curchunk *chunk
	for i := onum(0); i <= maxonum; i++ {
		if loc, ok := objectPositions[i]; ok {
			if curchunk == nil {
				curchunk = &chunk{
					startOnum: i,
				}
			}
			curchunk.positions = append(curchunk.positions, int64(loc))
		} else {
			if curchunk == nil {
				// the PDF might be corrupt
			} else {
				objectChunks = append(objectChunks, *curchunk)
				curchunk = nil
			}
		}
	}

	var str strings.Builder

	for _, chunk := range objectChunks {
		startOnum := chunk.startOnum
		fmt.Fprintf(&str, "%d %d\n", chunk.startOnum, len(chunk.positions))
		for i, pos := range chunk.positions {
			if int(startOnum)+i == 0 {
				fmt.Fprintf(&str, "%010d 65535 f \n", pos)
			} else {
				fmt.Fprintf(&str, "%010d 00000 n \n", pos)
			}
		}
	}

	fmt.Fprintln(&ret, "xref")
	fmt.Fprint(&ret, str.String())
	sum := fmt.Sprintf("%X", md5.Sum([]byte(str.String())))

	trailer := map[string]string{
		"/Size": fmt.Sprint(maxonum),
		"/Root": fmt.Sprintf("%d 0 R", rootObject),
		"/ID":   fmt.Sprintf("[<%s> <%s>]", sum, sum),
	}
	if infoObject != 0 {
		trailer["/Info"] = fmt.Sprintf("%d 0 R", infoObject)
	}
	fmt.Fprintln(&ret, "trailer <<")
	for k, v := range trailer {
		fmt.Fprintln(&ret, k, v)
	}
	fmt.Fprintf(&ret, ">>\nstartxref\n%d\n%%%%EOF\n", xrefpos)

	return ret.String(), nil
}

func fixXRefForFile(fn string) error {
	pdffile, err := os.Open(fn)
	if err != nil {
		return err
	}

	out, err := scan(pdffile)
	if err != nil {
		return err
	}
	pdffile.Close()
	if err = writePDFFile(fn, out); err != nil {
		return err
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("fixxref: expect file name of PDF file")
	}
	if err := fixXRefForFile(os.Args[1]); err != nil {
		log.Fatal(err)
	}
}
