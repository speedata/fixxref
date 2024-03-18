package scanner

import (
	"testing"
)

func TestSimple(t *testing.T) {
	str := `
%% Root
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
	for i, expected := range []int{0, 9, 61} {
		if got := p.objectPositions[onum(i)]; got != expected {
			t.Errorf("p.objectPositions[%d] = %d, want %d", i, got, expected)
		}
	}
}
