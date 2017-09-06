package edit

import (
	"errors"
	"io"

	mk "github.com/varyoo/mktorrent"

	"github.com/zeebo/bencode"
)

type (
	Torrent interface {
		Edit(Values)
		Save(io.Writer) error
		Read() Values
	}
	Values interface {
		Source() string
		Name() string
		Announce() []string
		Private() bool
	}
	meta struct {
		mk.Torrent
		mk.Info
	}
	multi struct {
		mk.TorrentMulti
	}
	single struct {
		mk.TorrentSingle
	}
)

func (t *multi) Edit(v Values) {
	write(&t.Torrent, &t.Info.Info, v)
}
func (t multi) Read() Values {
	return meta{t.Torrent, t.Info.Info}
}

func (t *single) Edit(v Values) {
	write(&t.Torrent, &t.Info.Info, v)
}
func (t single) Read() Values {
	return meta{t.Torrent, t.Info.Info}
}

func (v meta) Source() string {
	return v.Info.Source
}
func (v meta) Name() string {
	return v.Info.Name
}
func (v meta) Private() bool {
	return v.Info.Private == 1
}
func (v meta) Announce() []string {
	l := make([]string, 0, len(v.AnnounceList))
	for _, a := range v.AnnounceList {
		if len(a) == 1 {
			l = append(l, a[0])
		}
	}
	return l
}

func write(t *mk.Torrent, i *mk.Info, v Values) {
	i.Source = v.Source()
	i.Name = v.Name()
	t.AnnounceList = make([][]string, 0, len(v.Announce()))
	for _, a := range v.Announce() {
		t.AnnounceList = append(t.AnnounceList, []string{a})
	}
	if v.Private() {
		i.Private = 1
	} else {
		i.Private = 0
	}
}

func Load(r io.Reader) (Torrent, error) {
	d := bencode.NewDecoder(r)

	m := mk.TorrentMulti{}
	err := d.Decode(&m)
	if err == nil {
		return &multi{m}, nil
	}

	s := mk.TorrentSingle{}
	if err = d.Decode(&s); err == nil {
		return &single{s}, nil
	}

	return nil, errors.New("not a torrent file")
}
