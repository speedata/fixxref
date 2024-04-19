package scanner

import (
	"testing"
)

func TestSimple(t *testing.T) {
	str := `
1 0 obj
<<
	/Type /Catalog
	/Pages 2 0 R
>>
endobj

2 0 obj
<<
	/Type /Pages
	/Kids [ 3 0 R ]
	/Count 1
>>
endobj
`
	p, err := scanBody(str)
	if err != nil {
		t.Errorf("scanInternal got error, expect none: %s", err.Error())
	}
	for i, expected := range []int{0, 1, 53} {
		if got := p.objectPositions[onum(i)]; got != expected {
			t.Errorf("p.objectPositions[%d] = %d, want %d", i, got, expected)
		}
	}
	expected := 1
	if got := p.rootObject; got != onum(expected) {
		t.Errorf("p.rootObject = %d, want %d", got, expected)
	}
}
