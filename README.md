# autotorrent

~~~sh
GOPATH=/tmp
go get -u github.com/spf13/viper github.com/pkg/errors github.com/zeebo/bencode
cd mktorrent/autotorrent
go build
./autotorrent
~~~

## Usage

~~~
autotorrent [OPTIONS] PROFILE FILES...

Options:
  -h    show this help screen
  -v    be verbose
~~~

## Configuration file

The TOML configuration file is at `~/.config/autotorrent.toml`.
A configuration file is a collection of profiles.
The following is one profile:

~~~toml
[PROFILE]
announce = ["http://tracker.org:2710/announce", "udp://tracker.it:80"]
source = "YELLOW"
private = true
~~~
