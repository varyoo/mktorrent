package mktorrent

import (
	"crypto/sha1"
	"fmt"
	"github.com/pkg/errors"
	"github.com/zeebo/bencode"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func hashPiece(b []byte) []byte {
	h := sha1.New()
	h.Write(b)
	return h.Sum(nil)
}
func autoPieceLen(length int) (t int) {
	t = length / 1000
	if t < MinPieceLen {
		t = MinPieceLen
	} else if t > MaxPieceLen {
		t = MaxPieceLen
	}
	return
}

func MakeTorrent(path string, pieceLen int, source string, private bool, ann ...string) (*TorrentMulti, error) {
	t := &TorrentMulti{
		Torrent: Torrent{
			AnnounceList: make([][]string, 0),
			CreationDate: time.Now().Unix(),
			CreatedBy:    "varyoo",
		},
		Info: InfoMulti{
			Info: Info{
				Source: source,
				Name:   filepath.Base(path),
			},
		},
	}
	if private {
		t.Info.Private = 1
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// announce
	for _, a := range ann {
		t.AnnounceList = append(t.AnnounceList, []string{a})
	}

	// files
	var length int
	paths := make([]string, 0)
	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			f := File{
				Length: int(info.Size()),
				Path:   strings.Split(path, "/")[1:],
			}
			length += f.Length
			t.Info.Files = append(t.Info.Files, f)
			paths = append(paths, path)
		}
		return nil
	}

	if info.IsDir() {
		err = filepath.Walk(path, walker)
	} else {
		// Note the hack to have a leading root element.
		// Just like if it was called by filepath.Walk.
		err = walker(fmt.Sprintf("./%s", info.Name()), info, nil)
	}
	if err != nil {
		return nil, errors.Wrap(err, "exploring")
	}

	// piece length
	if pieceLen == 0 {
		pieceLen = autoPieceLen(length)
	}
	t.Info.PieceLength = pieceLen

	// hashing
	readers := make([]io.Reader, 0, len(paths))
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		readers = append(readers, f)
	}
	r := io.MultiReader(readers...)
	b := make([]byte, pieceLen)
	for {
		n, err := io.ReadFull(r, b)
		if err == io.ErrUnexpectedEOF {
			b = b[:n]
			err = nil
		} else if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "hashing")
		}
		t.Info.Pieces += string(hashPiece(b))
	}

	return t, nil
}