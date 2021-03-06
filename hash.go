package mktorrent

import (
	"crypto/sha1"
	"fmt"
	"io"
	"sync"
)

type piece struct {
	data []byte
	hash []byte
}

func hashRoutine(pieces <-chan piece, wg *sync.WaitGroup, pro Progress) {
	for piece := range pieces {
		hash := sha1.Sum(piece.data)
		copy(piece.hash, hash[:])
		pro.Increment()
	}
	wg.Done()
}

// Digest needs to be constructed by NewHash.
type Digest struct {
	intra      []byte
	piece      []byte
	piecesRead int64
	pieceCount int64
	pieceSize  int64
	pieces     chan piece
	wg         *sync.WaitGroup
	complete   string
	hash       []byte
	offset     int
	closed     bool
}

// String returns the hexadecimal encoding of the hash if complete. If not it returns "".
func (d *Digest) String() string {
	return d.complete
}

// Complete completes the hash first by appending to it the last irregular piece.
// Complete then returns the complete hash or error if any.
func (d *Digest) Complete() ([]byte, error) {
	if len(d.intra) != len(d.piece) {
		remaining := len(d.piece) - len(d.intra)
		d.piece = d.piece[:remaining]

		err := d.push()
		if err != nil {
			return nil, err
		}
	}

	close(d.pieces)
	d.closed = true
	d.wg.Wait() // wait for the hash to complete

	d.complete = fmt.Sprintf("%x", d.hash)
	return d.hash, nil
}

func NewHash(pieceSize, pieceCount int64, goroutines int, pro Progress) *Digest {
	if goroutines < 1 {
		goroutines = 1
	}
	buf := make([]byte, pieceSize)
	d := &Digest{
		piece:      buf,
		intra:      buf,
		pieceSize:  pieceSize,
		pieceCount: pieceCount,
		pieces:     make(chan piece, goroutines*2),
		wg:         &sync.WaitGroup{},
		hash:       make([]byte, pieceCount*sha1.Size),
	}
	for i := 0; i < goroutines; i++ {
		d.wg.Add(1)
		go hashRoutine(d.pieces, d.wg, pro)
	}
	return d
}

func (d *Digest) push() error {
	if d.piecesRead == d.pieceCount {
		return io.ErrShortBuffer
	}

	end := d.offset + sha1.Size
	d.pieces <- piece{data: d.piece, hash: d.hash[d.offset:end]}
	d.offset = end
	d.piecesRead++
	return nil
}

// ReadFrom hashes data from r until EOF or error.
//
// It returns the number of bytes read.
// Any error except io.EOF encountered during the read is also returned.
func (d *Digest) ReadFrom(r io.Reader) (int64, error) {
	var nt int64

	for {
		n, err := io.ReadFull(r, d.intra)
		nt += int64(n)
		if err != nil {
			d.intra = d.intra[n:]

			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nt, nil
			} else {
				return nt, err
			}
		}

		err = d.push()
		if err != nil {
			return nt, err
		}

		d.piece = make([]byte, d.pieceSize)
		d.intra = d.piece
	}
}

func (d *Digest) Close() error {
	if !d.closed {
		close(d.pieces)
		d.wg.Wait()
	}
	return nil
}
