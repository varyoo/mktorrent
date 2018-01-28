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

		length int `bencode:"-"` // cached length
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
		Save(io.Writer) error

		GetLength() int
		GetPieceLength() int
	}
)

func (i *Info) GetPieceLength() int {
	return i.PieceLength
}

func (i *InfoSingle) GetLength() int {
	return i.Length
}

func (i *InfoMulti) GetLength() (l int) {
	if i.length != 0 {
		return i.length
	}

	for _, f := range i.Files {
		l += f.Length
	}
	return l
}

func (i *Info) setPieces(p string) {
	i.Pieces = p
}

func (t *TorrentMulti) Save(w io.Writer) error {
	return bencode.NewEncoder(w).Encode(t)
}
func (t *TorrentMulti) Load(r io.Reader) error {
	return bencode.NewDecoder(r).Decode(t)
}

func (t *TorrentSingle) Save(w io.Writer) error {
	return bencode.NewEncoder(w).Encode(t)
}
func (t *TorrentSingle) Load(r io.Reader) error {
	return bencode.NewDecoder(r).Decode(t)
}

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

type preHashing struct {
	common
	paths       []string
	pieceLength int
	pieceCount  int
}

func (p *preHashing) GetPieceCount() int {
	return p.pieceCount
}

type Params struct {
	Path         string
	PieceLength  PieceLength
	Source       string
	Private      bool
	AnnounceList []string
}

type common interface {
	Buffer

	setPieces(string)
}

func PreHashing(ps Params) (preHashing, error) {
	pre := preHashing{}
	info := InfoMulti{}
	info.Name = filepath.Base(ps.Path)
	info.Files = make([]File, 0)
	info.Source = ps.Source
	if ps.Private {
		info.Private = 1
	}
	minDepth := 1
	size := 0

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
			info.Files = append(info.Files, f)
			pre.paths = append(pre.paths, path)
			size += f.Length
		}

		return nil
	}

	err := filepath.Walk(ps.Path, walker)
	if err != nil {
		return pre, err
	}

	info.length = size

	info.PieceLength = ps.PieceLength(size)
	pre.pieceCount = size / info.PieceLength
	if size%info.PieceLength != 0 {
		pre.pieceCount += 1
	}

	torrent := Torrent{
		AnnounceList: make([][]string, 0, len(ps.AnnounceList)),
		CreationDate: time.Now().Unix(),
		CreatedBy:    "varyoo",
	}
	for _, a := range ps.AnnounceList {
		torrent.AnnounceList = append(torrent.AnnounceList, []string{a})
	}

	if len(info.Files) > 1 {
		info.Name = info.Files[0].Path[minDepth-1]
		for j := 0; j < len(info.Files); j++ {
			f := &info.Files[j]
			f.Path = f.Path[minDepth:]
		}
		pre.common = &TorrentMulti{
			InfoMulti: info,
			Torrent:   torrent,
		}
	} else if len(info.Files) == 1 {
		pre.common = &TorrentSingle{
			InfoSingle: InfoSingle{
				Info:   info.Info,
				Length: info.Files[0].Length,
			},
			Torrent: torrent,
		}
	} else {
		return pre, errors.New("0 files torrent")
	}

	return pre, nil
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

var NoProgress = &noProgress{}

func (pre preHashing) MakeTorrent(goroutines int, pro Progress) (Buffer, error) {
	h := NewHash(pre.GetPieceLength(), pre.GetPieceCount(), goroutines, pro)
	defer h.Close()

	err := feed(h, pre.paths)
	if err != nil {
		return nil, err
	}

	hashBytes, err := h.Complete()
	if err != nil {
		return nil, err
	}

	pre.setPieces(string(hashBytes))
	return pre, nil
}

func feed(h *digest, files []string) error {
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
