package edit

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	mk "github.com/varyoo/mktorrent"
	"github.com/varyoo/mktorrent/test"
)

type v struct {
}

func (v v) Source() string {
	return "another source"
}
func (v v) Name() string {
	return "another name"
}
func (v v) Private() bool {
	return false
}
func (v v) Announce() []string {
	return []string{"another", "announce"}
}

func equal(a, b Values) bool {
	if len(a.Announce()) != len(b.Announce()) {
		return false
	}
	for i, a := range a.Announce() {
		if a != b.Announce()[i] {
			return false
		}
	}
	return a.Source() == b.Source() && a.Name() == b.Name() && a.Private() == b.Private()
}
func testEqual(t *testing.T, want, have Values) {
	if !equal(want, have) {
		t.Errorf("want: %s\nhave:%s", sprint(want), sprint(have))
	}
}
func sprint(r Values) string {
	return fmt.Sprintf("%s %s %s %t", r.Source(), r.Name(),
		strings.Join(r.Announce(), ":"), r.Private())
}

func Test(t *testing.T) {
	err := test.WriteFiles()
	if err != nil {
		t.Fatal(err)
	}
	tor, err := mk.MakeTorrent(test.Dir, 0, "yellow", true, "http://localhost/announce")
	if err != nil {
		t.Fatal(err)
	}
	b := &bytes.Buffer{}
	err = tor.Save(b)
	if err != nil {
		t.Fatal(err)
	}
	re, err := Load(b)
	if err != nil {
		t.Fatal(err)
	}
	re.Edit(v{})
	testEqual(t, v{}, re.Read())
	b = &bytes.Buffer{}
	re.Save(b)
	re, err = Load(b)
	if err != nil {
		t.Fatal(err)
	}
	testEqual(t, v{}, re.Read())
}
