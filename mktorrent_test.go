package mktorrent

import (
	"fmt"
	"os"
	"testing"

	"github.com/varyoo/mktorrent/test"
)

func init() {
	err := test.WriteFiles()
	if err != nil {
		panic(err)
	}
}

func Test(t *testing.T) {
	tor, err := MakeTorrent(test.Dir, 0, "yellow", true, "http://localhost/announce")
	if err != nil {
		t.Fatal(err)
		return
	}
	f, err := os.Create("test.torrent")
	if err != nil {
		t.Fatal(err)
		return
	}
	if err := tor.Save(f); err != nil {
		t.Fatal(err)
		return
	}
	f.Seek(0, 0)
	have := &TorrentMulti{}
	if err := have.Load(f); err != nil {
		t.Fatal(err)
		return
	}
	fmt.Printf("%+v\n", have)
}
