package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/varyoo/mktorrent"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	usage  string = `Usage: autotorrent [OPTIONS] PROFILE FILES...`
	config        = `Configuration file:
The TOML configuration file is at ~/.config/autotorrent.toml.
A configuration file is a collection of profiles.
The following is one profile:

[PROFILE]
announce = ["http://tracker.org:2710/announce", "udp://tracker.it:80"]
source = "YELLOW"
private = true`
)

func printHelp() {
	fmt.Printf("%s\n\nOptions:\n", usage)
	flag.PrintDefaults()
	fmt.Printf("\n%s\n", config)
}

var (
	verbose bool
	help    bool
)

func init() {
	flag.BoolVar(&help, "h", false, "show this help screen")
	flag.BoolVar(&verbose, "v", false, "be verbose")
}

type tor interface {
	Save(io.Writer) error
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
	profile := paths[0]
	paths = paths[1:]

	viper.SetConfigName("autotorrent")
	viper.AddConfigPath("$HOME/.config/")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if !viper.InConfig(profile) {
		return errors.New("profile not found")
	}
	ann := viper.GetStringSlice(fmt.Sprintf("%s.announce", profile))
	source := viper.GetString(fmt.Sprintf("%s.source", profile))
	private := viper.GetBool(fmt.Sprintf("%s.private", profile))

	if verbose {
		fmt.Printf("Profile: %s\nAnnounce:", profile)
		for _, a := range ann {
			fmt.Printf(" %s", a)
		}
		fmt.Printf("\nSource: %s\nPrivate: %t\n", source, private)
	}

	for _, path := range paths {
		if err := func() error {
			info, err := os.Stat(path)
			if err != nil {
				return errors.Wrap(err, "is it a dir or is it a file?")
			}

			var t tor
			var mode string
			if info.IsDir() {
				mode = "multiple files torrent"
				t, err = mktorrent.MakeMultiTorrent(path, 0, source, private, ann...)
			} else {
				mode = "single file torrent"
				t, err = mktorrent.MakeSingleTorrent(path, 0, source, private, ann...)
			}
			if verbose {
				fmt.Printf("Mode: %s\n", mode)
			}
			if err != nil {
				return errors.Wrap(err, "can't make torrent")
			}

			w, err := os.Create(fmt.Sprintf("%s.torrent", path))
			if err != nil {
				// clear enough
				return err
			}

			return errors.Wrap(t.Save(w), "can't save torrent")
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
