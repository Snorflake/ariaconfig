package ariaconfig

import "testing"

var teststr = `{{
aria = "cute"
u = "gay"
on = true
escaped = "wew \"lad\""
hex1 = 0xFF7700
hex2 = 0xFF
float = 0.56
}}`

//TestEverything tests everything
func TestEverything(t *testing.T) {
	p := newParser(teststr)
	if p == nil {
		t.Fatal("parser didn't initialize correctly")
	}

	_, err := p.parse() // get rid of first left itemLeftMeta
	if err != nil {
		t.Fatal("parser failed to parse leftmeta")
	}

	for stmt, err := p.parse(); ; stmt, err = p.parse() {
		if err != nil {
			t.Fatalf("Unexpected error: %v", err.Error())
		} else if stmt == nil {
			break //we hit right meta and nothing broke
		}
	}
}
