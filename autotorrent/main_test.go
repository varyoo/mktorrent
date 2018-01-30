package main

import (
	"flag"
	"io/ioutil"
	"os"
	"testing"

	mk "github.com/varyoo/mktorrent"
	"github.com/varyoo/mktorrent/test"
)

func init() {
	err := test.WriteFiles()
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("autotorrent.toml", []byte(`
[[profile]]
name = "green"
announce = ["http://localhost/announce"]
source = "GREEN"
private = true
max_piece_length = "16 mb"

[[profile]]
name = "yellow"
announce = ["http://localhost/announce", "udp://localhost:3000"]
source = "YELLOW"
private = false
`), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func TestSingle(t *testing.T) {
	err := try(flag.NewFlagSet("autotorrent", flag.ExitOnError), "-v", "green", test.File)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(test.File + ".torrent")
	if err != nil {
		t.Fatal(err)
	}

	tor := mk.Torrent{}
	err = tor.ReadFrom(f)
	if err != nil {
		t.Fatal(err)
	}

	if s := tor.Info.Source; s != "GREEN" {
		t.Errorf("Source is %s instead of GREEN", s)
	}
	if !tor.Info.Private {
		t.Error("Must be private")
	}

	if len(tor.AnnounceList) != 1 || len(tor.AnnounceList[0]) != 1 ||
		tor.AnnounceList[0][0] != "http://localhost/announce" {
		t.Error("Bad announce")
	}
}

func TestMulti(t *testing.T) {
	err := try(flag.NewFlagSet("autotorrent", flag.ExitOnError),
		"-g", "4", "yellow", test.Dir)
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(test.Dir + ".torrent")
	if err != nil {
		t.Fatal(err)
	}

	tor := mk.Torrent{}
	err = tor.ReadFrom(f)
	if err != nil {
		t.Fatal(err)
	}

	if s := tor.Info.Source; s != "YELLOW" {
		t.Error(s)
	}
	if tor.Info.Private {
		t.Error("Must be public")
	}

	m := map[string]bool{
		"http://localhost/announce": true,
		"udp://localhost:3000":      true,
	}
	if len(tor.AnnounceList) != 2 {
		t.Error("Bad announce")
	}
	for _, a := range tor.AnnounceList {
		if len(a) != 1 {
			t.Error("Bad announce")
		}
		delete(m, a[0])
	}
	if len(m) != 0 {
		t.Error("Bad announce")
	}
}
