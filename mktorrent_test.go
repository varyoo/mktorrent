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
	infoEqual(t, a.InfoSingle.Info, b.InfoSingle.Info)
	torrentEqual(t, a.Torrent, b.Torrent)
}

func multiEqual(t *testing.T, a, b *TorrentMulti) {
	infoEqual(t, a.InfoMulti.Info, b.InfoMulti.Info)
	torrentEqual(t, a.Torrent, b.Torrent)

	m := make(map[string]File)
	for _, f := range a.InfoMulti.Files {
		m[filepath.Join(f.Path...)] = f
	}
	for _, f := range b.InfoMulti.Files {
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
		InfoSingle: InfoSingle{
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
	err = wt.Save(b)
	if err != nil {
		t.Fatal(err)
	}
	have := &TorrentSingle{}
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
		InfoMulti: InfoMulti{
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
	err = wt.Save(b)
	if err != nil {
		t.Fatal(err)
	}
	have := &TorrentMulti{}
	err = have.Load(b)
	if err != nil {
		t.Fatal(err)
	}
	multiEqual(t, want, have)
}
