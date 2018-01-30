# mktorrent

[![GoDoc](https://godoc.org/github.com/varyoo/mktorrent?status.svg)](https://godoc.org/github.com/varyoo/mktorrent)

Everything you need to make torrent files in pure Go.

# autotorrent

`~/.config/autotorrent.toml` will be loaded automatically:

~~~ toml
[[profile]]
name = "green"
announce = ["http://localhost/announce"]
source = "GREEN"
private = true
max_piece_length = "16 mb"

[[profile]]
name = "yellow"
announce = ["http://localhost/announce", "udp://localhost:3000"]
source = "YELLOW"
private = false
~~~

Create a torrent with profile "green" settings:

~~~ sh
$ autotorrent green file/or/directory
 473 / 1100 [================>----------------------]  43.00% 49s
~~~

# Quick start

~~~ sh
GOPATH=/tmp
go get -u github.com/varyoo/bencode \
    github.com/cheggaaa/pb \
    github.com/varyoo/mktorrent \
    github.com/BurntSushi/toml \
    github.com/c2h5oh/datasize \
    github.com/pkg/errors \
    github.com/varyoo/mktorrent
cd mktorrent/autotorrent
go build
./autotorrent
~~~
