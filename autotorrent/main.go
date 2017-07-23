package main

import (
	"flag"
	"fmt"
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

func try() error {
	var v bool
	var h bool
	flag.BoolVar(&h, "h", false, "show this help screen")
	flag.BoolVar(&v, "v", false, "be verbose")
	flag.Parse()
	if h {
		printHelp()
		return nil
	}
	paths := flag.Args()
	if len(paths) < 2 {
		printHelp()
		return errors.New("not enough arguments")
	}
	tk := paths[0]
	paths = paths[1:]

	viper.SetConfigName("autotorrent")
	viper.AddConfigPath("$HOME/.config/")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if !viper.InConfig(tk) {
		return errors.New("profile not found")
	}
	ann := viper.GetStringSlice(fmt.Sprintf("%s.announce", tk))
	source := viper.GetString(fmt.Sprintf("%s.source", tk))
	private := viper.GetBool(fmt.Sprintf("%s.private", tk))

	for _, path := range paths {
		if err := func() error {
			t, err := mktorrent.MakeTorrent(path, 0, source, private, ann...)
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
