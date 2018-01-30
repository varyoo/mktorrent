package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cheggaaa/pb"
	"github.com/varyoo/mktorrent"

	"github.com/BurntSushi/toml"
	"github.com/c2h5oh/datasize"
	"github.com/pkg/errors"
)

const (
	usage string = `Usage: autotorrent [OPTIONS] PROFILE FILES...
More help at https://github.com/varyoo/mktorrent
`
)

func printHelp() {
	fmt.Printf("%s\nOPTIONS:\n", usage)
	flag.PrintDefaults()
}

var (
	verbose    bool
	help       bool
	goroutines int
)

func init() {
	flag.BoolVar(&help, "h", false, "show this help screen")
	flag.BoolVar(&verbose, "v", false, "be verbose")
	flag.IntVar(&goroutines, "g", 2, "number of Goroutines for calculating hashes")
}

type (
	profile struct {
		Name           string
		Announce       []string
		Source         string
		Private        bool
		MaxPieceLength string `toml:"max_piece_length"`
	}
	config struct {
		profile
		Profiles []profile `toml:"profile"`
	}
)

func (c config) GetProfile(name string) *profile {
	for _, p := range c.Profiles {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

func try() error {
	flag.Parse()
	if help {
		printHelp()
		return nil
	}
	paths := flag.Args()
	if len(paths) < 2 {
		printHelp()
		return errors.New("not enough arguments")
	}
	profileName := paths[0]
	paths = paths[1:]

	f, err := os.Open("autotorrent.toml")
	if err != nil {
		if os.IsNotExist(err) {
			f, err = os.Open(filepath.Join(os.Getenv("HOME"), ".config", "autotorrent.toml"))
		}
	}
	if err != nil {
		return errors.Wrap(err, "config file")
	}

	conf := config{}
	_, err = toml.DecodeReader(f, &conf)
	if err != nil {
		return errors.Wrap(err, "config")
	}

	pro := conf.GetProfile(profileName)
	if pro == nil {
		return errors.New("profile not found")
	}

	params := mktorrent.Params{
		AnnounceList: pro.Announce,
		Source:       pro.Source,
		Private:      pro.Private,
	}
	if pro.MaxPieceLength != "" {
		var bs datasize.ByteSize
		if err := bs.UnmarshalText([]byte(pro.MaxPieceLength)); err != nil {
			return errors.Wrapf(err, "%s.max_piece_length", profileName)
		}
		if bs.Bytes() > uint64(mktorrent.MaxPieceLen) {
			return fmt.Errorf("%s.max_piece_length is too big", profileName)
		}
		params.PieceLength = mktorrent.MaxPieceLength(int(bs.Bytes()))
		if verbose {
			fmt.Printf("Max piece length: %s\n", pro.MaxPieceLength)
		}
	} else {
		params.PieceLength = mktorrent.AutoPieceLength
	}

	if verbose {
		fmt.Printf("Profile: %s\nAnnounce:", profileName)
		for _, a := range params.AnnounceList {
			fmt.Printf(" %s", a)
		}
		fmt.Printf("\nSource: %s\nPrivate: %t\n", params.Source, params.Private)
	}

	for _, path := range paths {
		if err := func() error {
			params.Path = path
			fs, err := mktorrent.NewFilesystem(params)
			if err != nil {
				return errors.Wrap(err, "pre hashing")
			}

			pro := pb.New(fs.PieceCount).Start()
			wt, err := fs.MakeTorrent(goroutines, pro)
			if err != nil {
				return errors.Wrap(err, "hashing")
			}
			pro.Finish()

			w, err := os.Create(fmt.Sprintf("%s.torrent", path))
			if err != nil {
				// clear enough
				return err
			}

			return errors.Wrap(wt.WriteTo(w), "can't save torrent")
		}(); err != nil {
			return errors.Wrapf(err, "%s", path)
		}
	}
	return nil
}

func main() {
	if err := try(); err != nil {
		log.Println("failure:", err)
	}
}
