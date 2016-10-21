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
		t.Error("parser didn't initialize correctly")
		t.FailNow()
	}

	_, err := p.parse() // get rid of first left itemLeftMeta
	if err != nil {
		t.Error("parser failed to parse leftmeta")
		t.FailNow()
	}

	for stmt, err := p.parse(); ; stmt, err = p.parse() {
		if err != nil {
			t.Errorf("Unexpected error: %v", err.Error())
			t.FailNow()
		} else if stmt == nil {
			break //we hit right meta and nothing broke
		}
	}
}
