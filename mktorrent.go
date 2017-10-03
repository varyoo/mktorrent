package mktorrent

import (
	"crypto/sha1"
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

func MakeMultiTorrent(path string, pieceLen int, source string, private bool, ann ...string) (
	*TorrentMulti, error) {

	t := TorrentMulti{
		Torrent: mktorrent(ann),
		Info: InfoMulti{
			Info: mkinfo(source, path, private),
		},
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, errors.New("not a directory")
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

	err = filepath.Walk(path, walker)
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

	pieces, err := hash(r, pieceLen)
	t.Info.Pieces = pieces

	return &t, nil
}

func mktorrent(ann []string) Torrent {
	t := Torrent{
		AnnounceList: make([][]string, 0),
		CreationDate: time.Now().Unix(),
		CreatedBy:    "varyoo",
	}
	for _, a := range ann {
		t.AnnounceList = append(t.AnnounceList, []string{a})
	}
	return t
}

func mkinfo(source, path string, private bool) Info {
	i := Info{
		Source: source,
		Name:   filepath.Base(path),
	}
	if private {
		i.Private = 1
	}
	return i
}

func hash(r io.Reader, pieceLen int) (string, error) {
	b := make([]byte, pieceLen)
	var pieces string

	for {
		n, err := io.ReadFull(r, b)
		if err == io.ErrUnexpectedEOF {
			b = b[:n]
			err = nil
		} else if err == io.EOF {
			break
		}

		if err != nil {
			return "", err
		}
		pieces += string(hashPiece(b))
	}

	return pieces, nil
}

func MakeSingleTorrent(path string, pieceLen int, source string, private bool, ann ...string) (
	*TorrentSingle, error) {

	t := TorrentSingle{
		Torrent: mktorrent(ann),
		Info: InfoSingle{
			Info: mkinfo(source, path, private),
		},
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	length := int(info.Size())
	if pieceLen == 0 {
		pieceLen = autoPieceLen(length)
	}

	pieces, err := hash(f, pieceLen)
	if err != nil {
		return nil, err
	}

	t.Info.Pieces = pieces
	t.Info.PieceLength = pieceLen
	t.Info.Length = length

	return &t, nil
}
