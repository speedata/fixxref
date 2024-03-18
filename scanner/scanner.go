package scanner

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	startObj = regexp.MustCompile(`^(\d+) 0 obj$`)
)

type onum int

type pdf struct {
	xrefpos         int
	objectPositions map[onum]int
	infoObject      onum
	rootObject      onum
	maxOnum         onum
	body            string
}

func scanInternal(str string) (*pdf, error) {
	sr := strings.NewReader(str)
	sc := bufio.NewScanner(sr)
	p := &pdf{
		objectPositions: make(map[onum]int),
	}

	var ret strings.Builder
	pos := 0
	var nextObjIsInfo, nextObjIsRoot bool

	sc.Split(bufio.ScanLines)

	p.objectPositions[0] = 0
	for sc.Scan() {
		line := sc.Text()
		if res := startObj.FindAllStringSubmatch(line, -1); len(res) > 0 {
			objnum, err := strconv.Atoi(res[0][1])
			if err != nil {
				return p, err
			}
			if nextObjIsInfo {
				p.infoObject = onum(objnum)
				nextObjIsInfo = false
			} else if nextObjIsRoot {
				p.rootObject = onum(objnum)
				nextObjIsRoot = false
			}
			p.objectPositions[onum(objnum)] = pos
			p.maxOnum = onum(objnum) + 1
			fmt.Fprintln(&ret, line)
		} else if strings.HasPrefix(line, "xref") {
			// found startxref, remove everything from here
			p.xrefpos = pos
			break
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
	p.xrefpos = pos
	p.body = ret.String()
	return p, nil
}

// Scan reads the lines until the xref is found or until the end, whichever comes first. Objects with a comment
//
//	%% Info or %% Root
//
// right before the object number will be treated as info or root objects in the trailer.
//
// The return string is a complete PDF file which replaces the input.
func Scan(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	p, err := scanInternal(string(data))
	if err != nil {
		return "", err
	}

	type chunk struct {
		startOnum onum
		positions []int64
	}
	objectChunks := []chunk{}
	var curchunk *chunk
	for i := onum(0); i <= p.maxOnum; i++ {
		if loc, ok := p.objectPositions[i]; ok {
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
	str.WriteString(p.body)
	str.WriteString("xref\n")

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

	sum := fmt.Sprintf("%X", md5.Sum([]byte(str.String())))

	trailer := map[string]string{
		"/Size": fmt.Sprint(p.maxOnum),
		"/Root": fmt.Sprintf("%d 0 R", p.rootObject),
		"/ID":   fmt.Sprintf("[<%s> <%s>]", sum, sum),
	}
	if p.infoObject != 0 {
		trailer["/Info"] = fmt.Sprintf("%d 0 R", p.infoObject)
	}
	fmt.Fprintln(&str, "trailer <<")
	for k, v := range trailer {
		fmt.Fprintln(&str, k, v)
	}
	fmt.Fprintf(&str, ">>\nstartxref\n%d\n%%%%EOF\n", p.xrefpos)

	return str.String(), nil
}
