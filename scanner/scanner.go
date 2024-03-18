package scanner

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	startObj = regexp.MustCompile(`(?ms)^(\d+) 0 obj.*?$.*?endobj.*?$`)
	isRoot   = regexp.MustCompile(`/Type\s*/Catalog\W`)
	streamRe = regexp.MustCompile(`(?ms)stream\n(.*?)^endstream`)
	lengthRe = regexp.MustCompile(`/Length\s*?(\d+)`)
)

type onum int

type pdf struct {
	objectPositions map[onum]int
	infoObject      onum
	rootObject      onum
	maxOnum         onum
	body            strings.Builder
}

// ~~> beforeObject "%PDF-1.6\n%···\n\n1 0 obj\n<<\n    /Type /Catalog\n "
func scanBody(str string) (*pdf, error) {
	p := &pdf{
		objectPositions: make(map[onum]int),
	}
	p.objectPositions[0] = 0
	pos := 0
	for {
		idx := startObj.FindStringSubmatchIndex(str)
		if len(idx) == 0 {
			break
		}
		beforeObject := str[0:idx[0]]
		p.body.WriteString(beforeObject)
		on, err := strconv.Atoi(str[idx[2]:idx[3]])
		if err != nil {
			return nil, err
		}
		objectNumber := onum(on)
		p.objectPositions[objectNumber] = pos + idx[0]
		pos += idx[1]

		objString := str[idx[0]:idx[1]]

		// root object if it contains /Type /Catalog
		if isRoot.MatchString(objString) {
			p.rootObject = objectNumber
		}

		// a stream? Then update the /Length
		streamSubmatch := streamRe.FindStringSubmatch(objString)
		if len(streamSubmatch) > 0 {
			streamLength := len(streamSubmatch[1])
			objString = lengthRe.ReplaceAllString(objString, fmt.Sprintf("/Length %d", streamLength))
		}

		p.body.WriteString(objString)
		if p.maxOnum < objectNumber {
			p.maxOnum = objectNumber
		}

		str = str[idx[1]:]
	}
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
	p, err := scanBody(string(data))
	if err != nil {
		return "", err
	}

	type chunk struct {
		startOnum onum
		positions []int64
	}
	objectChunks := []chunk{}
	var curchunk *chunk
	for i := onum(0); i <= p.maxOnum+1; i++ {
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
	p.body.WriteString("\n")
	xrefpos := p.body.Len()
	p.body.WriteString("xref\n")

	for _, chunk := range objectChunks {
		startOnum := chunk.startOnum
		fmt.Fprintf(&p.body, "%d %d\n", chunk.startOnum, len(chunk.positions))
		for i, pos := range chunk.positions {
			if int(startOnum)+i == 0 {
				fmt.Fprintf(&p.body, "%010d 65535 f \n", pos)
			} else {
				fmt.Fprintf(&p.body, "%010d 00000 n \n", pos)
			}
		}
	}

	fmt.Fprintln(&p.body, "trailer <<")
	fmt.Fprintln(&p.body, "    /Size", p.maxOnum+1)
	fmt.Fprintln(&p.body, "    /Root", fmt.Sprintf("%d 0 R", p.rootObject))
	if p.infoObject != 0 {
		fmt.Fprintln(&p.body, "    /Info", fmt.Sprintf("%d 0 R", p.infoObject))
	}
	fmt.Fprintf(&p.body, ">>\nstartxref\n%d\n%%%%EOF\n", xrefpos)
	return p.body.String(), nil
}
