package mktorrent

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/zeebo/bencode"
)

const (
	AutoMinPieceLength int64 = 16384    // 2^14
	AutoMaxPieceLength       = 67108864 // 2^26
)

type (
	Info struct {
		PieceLength int64  `bencode:"piece length"`
		Pieces      string `bencode:"pieces"`
		Source      string `bencode:"source,omitempty"`
		Private     bool   `bencode:"private,omitempty"`
		Name        string `bencode:"name"`

		Files       []File `bencode:"files,omitempty"`  // multi-file mode only
		TotalLength int64  `bencode:"length,omitempty"` // single-file mode only
	}
	File struct {
		Length int64    `bencode:"length"`
		Path   []string `bencode:"path"`
	}
	Torrent struct {
		AnnounceList [][]string `bencode:"announce-list,omitempty"`
		Announce     string     `bencode:"announce,omitempty"`
		CreationDate int64      `bencode:"creation date,omitempty"`
		Comment      string     `bencode:"comment,omitempty"`
		CreatedBy    string     `bencode:"created by,omitempty"`
		UrlList      string     `bencode:"url-list,omitempty"`

		Info `bencode:"info"`
	}

	Buffer interface {
		WriteTo(io.Writer) error
	}
)

type PieceLength func(length int64) int64

func BoundPieceLength(min, max int64) PieceLength {
	return func(length int64) (t int64) {
		t = length / 1000
		if t < min {
			t = min
		} else if t > max {
			t = max
		}
		return
	}
}

var AutoPieceLength = BoundPieceLength(AutoMinPieceLength, AutoMaxPieceLength)

func MaxPieceLength(max int64) PieceLength {
	return BoundPieceLength(AutoMinPieceLength, max)
}

func (t *Torrent) WriteTo(w io.Writer) error {
	if len(t.Files) > 1 {
		// shouldn't be a problem anyway
		t.TotalLength = 0
	}
	return bencode.NewEncoder(w).Encode(t)
}

func (t *Torrent) ReadFrom(r io.Reader) error {
	return bencode.NewDecoder(r).Decode(t)
}

// Filesystem can make torrents from your files or directories.
type Filesystem struct {
	Torrent

	PieceCount int64
	RealPaths  []string
}

func (fs *Filesystem) NewHash(goroutines int, pro Progress) *Digest {
	return NewHash(fs.PieceLength, fs.PieceCount, goroutines, pro)
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
	var size int64
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
				Length: fi.Size(),
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

	if n := len(files); n > 1 {
		for j := 0; j < len(files); j++ {
			f := &files[j]
			f.Path = f.Path[minDepth:]
		}
	} else if n == 1 {
		// create a single-file mode torrent
		files = nil
	} else {
		return nil, errors.New("no file found")
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
		Files:       files,
		TotalLength: size,
		Private:     ps.Private,
	}

	torrent := Torrent{
		AnnounceList: make([][]string, 0, len(ps.AnnounceList)),
		CreationDate: time.Now().Unix(),
		CreatedBy:    "varyoo",
		Info:         info,
	}
	for _, a := range ps.AnnounceList {
		torrent.AnnounceList = append(torrent.AnnounceList, []string{a})
	}

	fs := &Filesystem{
		Torrent:    torrent,
		RealPaths:  realPaths,
		PieceCount: pieceCount,
	}
	return fs, nil
}

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

func (fs Filesystem) MakeTorrent(goroutines int, pro Progress) (Buffer, error) {
	// Filesystem is passed by value

	h := fs.NewHash(goroutines, pro)
	defer h.Close()

	err := feed(h, fs.RealPaths)
	if err != nil {
		return nil, err
	}

	hashBytes, err := h.Complete()
	if err != nil {
		return nil, err
	}

	fs.Info.Pieces = string(hashBytes)
	return &fs, nil
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
