package mktorrent

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/varyoo/mktorrent/test"
)

func init() {
	err := test.WriteFiles()
	if err != nil {
		panic(err)
	}
}

func torrentEqual(t *testing.T, a, b Torrent) {
	m := make(map[string]bool)
	for _, row := range a.AnnounceList {
		if len(row) != 1 {
			t.Fatal()
		}
		m[row[0]] = true
	}
	for _, row := range b.AnnounceList {
		if len(row) != 1 {
			t.Fatal()
		}
		if !m[row[0]] {
			t.Error()
		}
		delete(m, row[0])
	}
	if len(m) != 0 {
		t.Error()
	}
}

func infoEqual(t *testing.T, a, b Info) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("hex %x", b.Pieces)
		t.Errorf("Have: %+v\nand: %+v\n", a, b)
	}
}

func singleEqual(t *testing.T, a, b *TorrentSingle) {
	infoEqual(t, a.Info.Info, b.Info.Info)
	torrentEqual(t, a.Torrent, b.Torrent)
}

func multiEqual(t *testing.T, a, b *TorrentMulti) {
	infoEqual(t, a.Info.Info, b.Info.Info)
	torrentEqual(t, a.Torrent, b.Torrent)

	m := make(map[string]File)
	for _, f := range a.Info.Files {
		m[filepath.Join(f.Path...)] = f
	}
	for _, f := range b.Info.Files {
		path := filepath.Join(f.Path...)
		f = m[path]
		if len(f.Path) == 0 {
			t.Error()
		}
		delete(m, path)
	}
	if len(m) != 0 {
		t.Error()
	}
}

func TestSingle(t *testing.T) {
	want := &TorrentSingle{
		Torrent: Torrent{
			AnnounceList: [][]string{
				{"http://localhost/announce"},
				{"udp://localhost:3000"},
			},
		},
		Info: InfoSingle{
			Info: Info{
				Source:      "green",
				Private:     1,
				Name:        filepath.Base(test.File),
				PieceLength: 16384,
				Pieces:      test.FilePieces,
			},
			Length: 0,
		},
	}
	have, err := MakeSingleTorrent(test.File, 0, "green", true, "http://localhost/announce",
		"udp://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	singleEqual(t, want, have)

	b := &bytes.Buffer{}
	err = have.Save(b)
	if err != nil {
		t.Fatal(err)
	}
	have = &TorrentSingle{}
	err = have.Load(b)
	if err != nil {
		t.Fatal(err)
	}
	singleEqual(t, want, have)
}

func TestMulti(t *testing.T) {
	want := &TorrentMulti{
		Torrent: Torrent{
			AnnounceList: [][]string{
				{"http://localhost/announce"},
				{"udp://localhost:3000"},
			},
		},
		Info: InfoMulti{
			Info: Info{
				Source:      "green",
				Private:     0,
				Name:        filepath.Base(test.Dir),
				PieceLength: 16384,
				Pieces:      test.DirPieces,
			},
			Files: []File{
				{Length: 0, Path: []string{"La douceur de l'ennui"}},
				{0, []string{"Les panthères modernistes"}},
				{0, []string{"Subdirectory", "Le meurtre moderniste"}},
			},
		},
	}
	have, err := MakeMultiTorrent(test.Dir, 0, "green", false, "http://localhost/announce",
		"udp://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	multiEqual(t, want, have)

	b := &bytes.Buffer{}
	err = have.Save(b)
	if err != nil {
		t.Fatal(err)
	}
	have = &TorrentMulti{}
	err = have.Load(b)
	if err != nil {
		t.Fatal(err)
	}
	multiEqual(t, want, have)
}
