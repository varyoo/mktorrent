package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cheggaaa/pb"
	"github.com/varyoo/mktorrent"

	"github.com/BurntSushi/toml"
	"github.com/c2h5oh/datasize"
	"github.com/pkg/errors"
)

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: autotorrent [OPTIONS] <profile> <target directory or filename>...\n\n"+
			"Options:\n")
	flag.PrintDefaults()
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

func try(flags *flag.FlagSet, args ...string) error {
	var verbose bool
	var goroutines int

	flags.Usage = usage
	flags.BoolVar(&verbose, "v", false, "be verbose")
	flags.IntVar(&goroutines, "g", 2, "number of Goroutines for calculating hashes")
	flags.Parse(args)

	paths := flags.Args()
	if len(paths) < 2 {
		usage()
		return errors.New("see usage")
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

	ps := mktorrent.Params{
		AnnounceList: pro.Announce,
		Source:       pro.Source,
		Private:      pro.Private,
	}
	if pro.MaxPieceLength != "" {
		id := fmt.Sprintf("%s.max_piece_length", profileName)

		var bs datasize.ByteSize
		if err := bs.UnmarshalText([]byte(pro.MaxPieceLength)); err != nil {
			return errors.Wrapf(err, id)
		}
		if bs.Bytes() > uint64(mktorrent.AutoMaxPieceLength) {
			return fmt.Errorf("%s is too big", id)
		}
		ps.PieceLength = mktorrent.MaxPieceLength(int64(bs.Bytes()))

		if verbose {
			fmt.Printf("Max piece length: %s\n", pro.MaxPieceLength)
		}
	} else {
		ps.PieceLength = mktorrent.AutoPieceLength
	}

	if verbose {
		fmt.Printf("Profile: %s\n"+
			"Announce: %s\n"+
			"Source: %s\n"+
			"Private: %t\n",
			profileName,
			strings.Join(ps.AnnounceList, ", "),
			ps.Source,
			ps.Private)
	}

	for _, path := range paths {
		if err := func() error {
			ps.Path = path
			fs, err := mktorrent.NewFilesystem(ps)
			if err != nil {
				return errors.Wrap(err, "pre hashing")
			}

			pro := pb.New64(fs.PieceCount).Start()
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
	log.SetFlags(0)
	if err := try(flag.CommandLine, os.Args[1:]...); err != nil {
		log.Fatalln("Failure:", err)
	}
}
