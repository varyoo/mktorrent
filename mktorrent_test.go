package mktorrent

import (
	"fmt"
	"os"
	"testing"
)

func Test(t *testing.T) {
	tor, err := MakeTorrent("./dir", 0, "yellow", true, "http://localhost/announce")
	if err != nil {
		t.Fatal(err)
		return
	}
	f, err := os.Create("dir.torrent")
	if err != nil {
		t.Fatal(err)
		return
	}
	if err := tor.Save(f); err != nil {
		t.Fatal(err)
		return
	}
	f.Seek(0, 0)
	have := &Torrent{}
	if err := have.Load(f); err != nil {
		t.Fatal(err)
		return
	}
	fmt.Printf("%+v\n", have)
}
