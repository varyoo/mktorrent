package test

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	Dir  string = "Test files"
	file string = "Les panthères modernistes"
)

var File = filepath.Join(Dir, file)
var FilePieces, DirPieces string

func init() {
	b, err := hex.DecodeString("6c6d92c46205e60700c7545ff7977dae854af76e")
	if err != nil {
		panic(err)
	}
	FilePieces = string(b)

	b, err = hex.DecodeString("2e4a5ac6ec3817ca4702a8ddc0f35133772eff35")
	if err != nil {
		panic(err)
	}
	DirPieces = string(b)
}

func WriteFiles() error {
	err := os.MkdirAll(filepath.Join(Dir, "Subdirectory"), os.ModePerm)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(File, []byte("Message court."), os.ModePerm)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(Dir, "La douceur de l'ennui"),
		[]byte("Pivot sur le toit."), os.ModePerm)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(Dir, "Subdirectory", "Le meurtre moderniste"),
		[]byte("Chasser l'âme."), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
