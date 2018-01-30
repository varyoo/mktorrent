package mktorrent

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/varyoo/bencode" // waiting for zeebo to merge bugfix
)

const (
	// 2^14: mktorrent minimum
	MinPieceLen int = 16384
	// 2^26: mktorrent maximum
	MaxPieceLen = 67108864
)

type (
	Info struct {
		PieceLength int    `bencode:"piece length"`
		Pieces      string `bencode:"pieces"`
		Source      string `bencode:"source,omitempty"`
		Private     int    `bencode:"private,omitempty"`
		Name        string `bencode:"name"`
	}
	InfoMulti struct {
		Info
		Files []File `bencode:"files"`
	}
	InfoSingle struct {
		Info
		Length int `bencode:"length"`
	}
	File struct {
		Length int      `bencode:"length"`
		Path   []string `bencode:"path"`
	}
	Torrent struct {
		AnnounceList [][]string `bencode:"announce-list,omitempty"`
		Announce     string     `bencode:"announce,omitempty"`
		CreationDate int64      `bencode:"creation date,omitempty"`
		Comment      string     `bencode:"comment,omitempty"`
		CreatedBy    string     `bencode:"created by,omitempty"`
		UrlList      string     `bencode:"url-list,omitempty"`
	}
	TorrentMulti struct {
		Torrent
		InfoMulti `bencode:"info"`
	}
	TorrentSingle struct {
		Torrent
		InfoSingle `bencode:"info"`
	}

	Buffer interface {
		WriteTo(io.Writer) error
	}
)

func BoundPieceLength(min, max int) PieceLength {
	return func(length int) (t int) {
		t = length / 1000
		if t < min {
			t = min
		} else if t > max {
			t = max
		}
		return
	}
}

var AutoPieceLength = BoundPieceLength(MinPieceLen, MaxPieceLen)

func MaxPieceLength(max int) PieceLength {
	return BoundPieceLength(MinPieceLen, max)
}

type TorrentEditor struct {
	Info
	Torrent

	Files      []File
	PieceCount int
	Length     int
}

func (t *TorrentEditor) NewHash(goroutines int, pro Progress) *Digest {
	return NewHash(t.PieceLength, t.PieceCount, goroutines, pro)
}

func (t *TorrentEditor) WriteTo(w io.Writer) error {
	var v interface{}
	if n := len(t.Files); n > 1 {
		v = &TorrentMulti{
			InfoMulti: InfoMulti{
				Files: t.Files,
				Info:  t.Info,
			},
			Torrent: t.Torrent,
		}
	} else if n == 1 {
		v = &TorrentSingle{
			InfoSingle: InfoSingle{
				Info:   t.Info,
				Length: t.Length,
			},
			Torrent: t.Torrent,
		}
	} else {
		panic("0 files torrent")
	}

	return bencode.NewEncoder(w).Encode(v)
}

// Filesystem can make torrents from your files or directories.
type Filesystem struct {
	TorrentEditor

	RealPaths []string
}

type Params struct {
	Path         string
	PieceLength  PieceLength
	Source       string
	Private      bool
	AnnounceList []string
}

func NewFilesystem(ps Params) (*Filesystem, error) {
	files := make([]File, 0)
	minDepth := 1
	size := 0
	realPaths := make([]string, 0)

	walker := func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fi.IsDir() {
			sp := strings.Split(path, string(os.PathSeparator))
			if len(sp) < 1 {
				return errors.Wrapf(err, "malformed path %s", path)
			}
			if depth := len(sp); depth < minDepth {
				minDepth = depth
			}

			f := File{
				Length: int(fi.Size()),
				Path:   sp,
			}
			files = append(files, f)
			realPaths = append(realPaths, path)
			size += f.Length
		}

		return nil
	}

	err := filepath.Walk(ps.Path, walker)
	if err != nil {
		return nil, err
	}

	for j := 0; j < len(files); j++ {
		f := &files[j]
		f.Path = f.Path[minDepth:]
	}

	pieceLength := ps.PieceLength(size)
	pieceCount := size / pieceLength
	if size%pieceLength != 0 {
		pieceCount += 1
	}

	info := Info{
		PieceLength: pieceLength,
		Source:      ps.Source,
		Name:        filepath.Base(ps.Path),
	}
	if ps.Private {
		info.Private = 1
	}

	torrent := Torrent{
		AnnounceList: make([][]string, 0, len(ps.AnnounceList)),
		CreationDate: time.Now().Unix(),
		CreatedBy:    "varyoo",
	}
	for _, a := range ps.AnnounceList {
		torrent.AnnounceList = append(torrent.AnnounceList, []string{a})
	}

	te := TorrentEditor{
		Info:       info,
		Torrent:    torrent,
		Files:      files,
		PieceCount: pieceCount,
		Length:     size,
	}
	fs := &Filesystem{
		TorrentEditor: te,
		RealPaths:     realPaths,
	}
	return fs, nil
}

type PieceLength func(length int) int

type (
	Progress interface {
		Increment() int
	}
	noProgress struct {
	}
)

func (n *noProgress) Increment() int {
	return 0
}

var NoProgress *noProgress = nil

func (f *Filesystem) MakeTorrent(goroutines int, pro Progress) (Buffer, error) {
	h := f.NewHash(goroutines, pro)
	defer h.Close()

	err := feed(h, f.RealPaths)
	if err != nil {
		return nil, err
	}

	hashBytes, err := h.Complete()
	if err != nil {
		return nil, err
	}

	f.Info.Pieces = string(hashBytes)
	return f, nil
}

func feed(h *Digest, files []string) error {
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			return err
		}

		_, err = h.ReadFrom(f)
		if err != nil {
			return err
		}

		err = f.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
