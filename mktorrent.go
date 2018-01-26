package mktorrent

import (
	"crypto/sha1"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/zeebo/bencode"
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
		Info InfoMulti `bencode:"info"`
	}
	TorrentSingle struct {
		Torrent
		Info InfoSingle `bencode:"info"`
	}

	Buffer interface {
		Save(io.Writer) error
	}
)

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

type file struct {
	size int
	path string
}

type PieceLength func(length int) int

type Params struct {
	Path         string
	PieceLength  PieceLength
	Source       string
	Private      bool
	AnnounceList []string
	Goroutines   int
}

func MakeTorrent(params Params) (Buffer, error) {
	var length int
	files := make([]file, 0)

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			f := file{
				size: int(info.Size()),
				path: path,
			}
			length += f.size
			files = append(files, f)
		}

		return nil
	}

	err := filepath.Walk(params.Path, walker)
	if err != nil {
		return nil, errors.Wrap(err, "exploring")
	}

	pieceLen := params.PieceLength(length)

	if params.Goroutines < 1 {
		params.Goroutines = 1
	}
	pieces := make(chan piece, params.Goroutines)
	wg := &sync.WaitGroup{}
	for i := 0; i < params.Goroutines; i++ {
		wg.Add(1)
		go hashRoutine(pieces, wg)
	}

	pieceCount := length / pieceLen
	if length%pieceLen != 0 {
		pieceCount += 1
	}
	hash := make([]byte, pieceCount*sha1.Size)

	err = feed(pieces, pieceLen, files, hash)
	if err != nil {
		return nil, errors.Wrap(err, "reading")
	}

	wg.Wait() // wait for the hash to complete

	torrent := Torrent{
		AnnounceList: make([][]string, 0, len(params.AnnounceList)),
		CreationDate: time.Now().Unix(),
		CreatedBy:    "varyoo",
	}
	for _, a := range params.AnnounceList {
		torrent.AnnounceList = append(torrent.AnnounceList, []string{a})
	}

	info := Info{
		Source:      params.Source,
		Name:        filepath.Base(params.Path),
		Pieces:      string(hash),
		PieceLength: pieceLen,
	}
	if params.Private {
		info.Private = 1
	}

	if n := len(files); n == 0 {
		return nil, errors.New("0 files, is this something possible?")
	} else if n == 1 {
		t := TorrentSingle{
			Torrent: torrent,
			Info: InfoSingle{
				Info:   info,
				Length: length,
			},
		}
		return &t, nil
	} else {
		t := TorrentMulti{
			Torrent: torrent,
			Info: InfoMulti{
				Info:  info,
				Files: make([]File, 0, len(files)),
			},
		}
		for _, file := range files {
			f := File{}
			path := strings.Split(file.path, string(os.PathSeparator))
			if len(path) < 1 {
				return nil, errors.Wrapf(err, "unexpected path %s", file.path)
			}
			f.Path = path[1:]
			f.Length = file.size
			t.Info.Files = append(t.Info.Files, f)
		}
		return &t, nil
	}
}

type piece struct {
	data []byte
	hash []byte
}

func hashRoutine(pieces <-chan piece, wg *sync.WaitGroup) {
	for piece := range pieces {
		hash := sha1.Sum(piece.data)
		copy(piece.hash, hash[:])
	}
	wg.Done()
}

func feed(pieces chan<- piece, pieceLen int, files []file, hash []byte) error {
	defer close(pieces)

	b := make([]byte, pieceLen) // piece buffer
	s := b                      // intra-piece buffer
	offset := 0                 // hash offset

	for _, meta := range files {
		f, err := os.Open(meta.path)
		if err != nil {
			return err
		}

		for {
			n, err := io.ReadFull(f, s)

			if n == len(s) {
				// this piece is filled up
				end := offset + sha1.Size
				pieces <- piece{data: b, hash: hash[offset:end]}

				b = make([]byte, pieceLen)
				s = b
				offset = end
			} else {
				s = s[n:]
			}

			if err == io.ErrUnexpectedEOF || err == io.EOF {
				if err = f.Close(); err != nil {
					return err
				}
				break // next file
			} else if err != nil {
				return err
			}
		}
	}

	if len(s) != len(b) {
		// finally append the hash of the last irregular piece to the hash string
		remaining := len(b) - len(s)
		pieces <- piece{data: b[:remaining], hash: hash[offset:]}
	}

	return nil
}
