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

func testEqual(t *testing.T, want, have *Torrent) {
	t.Helper()

	if !reflect.DeepEqual(want.Info, have.Info) {
		t.Fatalf("Want: %+v\nBut have: %+v", want.Info, have.Info)
	}

	// check files
	m := make(map[string]File)
	for _, f := range want.Files {
		m[filepath.Join(f.Path...)] = f
	}
	for _, f := range have.Files {
		path := filepath.Join(f.Path...)
		f = m[path]
		if len(f.Path) == 0 {
			t.Fatal()
		}
		delete(m, path)
	}
	if len(m) != 0 {
		t.Fatal()
	}

	// check announce-list
	ma := make(map[string]bool)
	for _, row := range want.AnnounceList {
		if len(row) != 1 {
			t.Fatal()
		}
		ma[row[0]] = true
	}
	for _, row := range have.AnnounceList {
		if len(row) != 1 {
			t.Fatal()
		}
		if !ma[row[0]] {
			t.Fatal()
		}
		delete(ma, row[0])
	}
	if len(ma) != 0 {
		t.Fatal()
	}
}

func TestSingle(t *testing.T) {
	want := &Torrent{
		AnnounceList: [][]string{
			{"http://localhost/announce"},
			{"udp://localhost:3000"},
		},
		Info: Info{
			Source:      "green",
			Private:     true,
			Name:        filepath.Base(test.File),
			PieceLength: 16384,
			Pieces:      test.FilePieces,
			TotalLength: 14,
		},
	}

	ps := Params{
		Path:        test.File,
		PieceLength: AutoPieceLength,
		Source:      "green",
		Private:     true,
		AnnounceList: []string{
			"http://localhost/announce",
			"udp://localhost:3000",
		},
	}
	fs, err := NewFilesystem(ps)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := fs.MakeTorrent(1, NoProgress)
	if err != nil {
		t.Fatal(err)
	}

	b := &bytes.Buffer{}
	err = wt.WriteTo(b)
	if err != nil {
		t.Fatal(err)
	}

	have := &Torrent{}
	err = have.ReadFrom(b)
	if err != nil {
		t.Fatal(err)
	}

	testEqual(t, want, have)
}

func TestMulti(t *testing.T) {
	want := &Torrent{
		AnnounceList: [][]string{
			{"http://localhost/announce"},
			{"udp://localhost:3000"},
		},
		Info: Info{
			Source:      "green",
			Private:     false,
			Name:        filepath.Base(test.Dir),
			PieceLength: 16384,
			Pieces:      test.DirPieces,
			Files: []File{
				{Length: 18, Path: []string{"La douceur de l'ennui"}},
				{14, []string{"Les panthères modernistes"}},
				{15, []string{"Subdirectory", "Le meurtre moderniste"}},
			},
		},
	}

	ps := Params{
		Path:        test.Dir,
		PieceLength: AutoPieceLength,
		Source:      "green",
		Private:     false,
		AnnounceList: []string{
			"http://localhost/announce",
			"udp://localhost:3000",
		},
	}
	fs, err := NewFilesystem(ps)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := fs.MakeTorrent(4, NoProgress)
	if err != nil {
		t.Fatal(err)
	}

	b := &bytes.Buffer{}
	err = wt.WriteTo(b)
	if err != nil {
		t.Fatal(err)
	}

	have := &Torrent{}
	err = have.ReadFrom(b)
	if err != nil {
		t.Fatal(err)
	}

	testEqual(t, want, have)
}
