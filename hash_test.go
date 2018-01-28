package mktorrent

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"testing"
)

func sumAgainst(t *testing.T, h *digest, pieces ...string) {
	var want string
	for _, piece := range pieces {
		a := sha1.Sum([]byte(piece))
		want += fmt.Sprintf("%x", a)
	}

	haveBytes, err := h.Complete()
	t.Helper()
	if err != nil {
		t.Fatal("Complete:", err)
	}
	have := fmt.Sprintf("%x", haveBytes)
	if want != have {
		t.Fatalf("Want hash: %s, but have: %s", want, have)
	}
}

func writeError(t *testing.T, d *digest, s string, mustFail bool) {
	_, err := d.ReadFrom(bytes.NewBuffer([]byte(s)))
	if err != nil && !mustFail {
		t.Helper()
		t.Fatal("ReadFrom:", err)
	}
}

func write(t *testing.T, d *digest, s string) {
	writeError(t, d, s, false)
}

func Test0(t *testing.T) {
	h := NewHash(1, 0, 1, NoProgress)
	sumAgainst(t, h)
}

func TestSingleIrregularPiece(t *testing.T) {
	h := NewHash(30, 1, 1, NoProgress)
	write(t, h, "12")
	sumAgainst(t, h, "12")
}

func TestIrregular(t *testing.T) {
	h := NewHash(2, 3, 1, NoProgress)
	write(t, h, "11")
	write(t, h, "22")
	write(t, h, "3")
	sumAgainst(t, h, "11", "22", "3")
}

func TestRegular(t *testing.T) {
	h := NewHash(2, 2, 1, NoProgress)

	// piece 1
	write(t, h, "1")
	write(t, h, "2")

	// piece 2
	write(t, h, "34")

	sumAgainst(t, h, "12", "34")
}

func TestBehavior(t *testing.T) {
	h := NewHash(1, 1, 0, NoProgress)
	write(t, h, "1")
	h.Close()
}

func TestOverflow(t *testing.T) {
	h := NewHash(0, 0, 1, NoProgress)
	writeError(t, h, "1", true)
}
