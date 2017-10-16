package mktorrent

import (
	"crypto/sha1"
	"io"
	"os"
	"path/filepath"
	"strings"
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

	WriteTorrent interface {
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

func mkTorrent(ann []string) Torrent {
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

func mkInfo(source, path string, private bool) Info {
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

type file struct {
	size int
	path string
}

func MakeTorrent(path string, pieceLen int, source string, private bool, announces []string) (
	WriteTorrent, error) {

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

	err := filepath.Walk(path, walker)
	if err != nil {
		return nil, errors.Wrap(err, "exploring")
	}

	if pieceLen == 0 {
		pieceLen = autoPieceLen(length)
	}

	readers := make([]io.Reader, 0, len(files))
	for _, file := range files {
		f, err := os.Open(file.path)
		if err != nil {
			return nil, err
		}
		readers = append(readers, f)
	}
	r := io.MultiReader(readers...)

	pieces, err := hash(r, pieceLen)
	if err != nil {
		return nil, errors.Wrap(err, "hashing")
	}

	torrent := mkTorrent(announces)
	info := mkInfo(source, path, private)
	info.Pieces = pieces
	info.PieceLength = pieceLen

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
