package test

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	Dir  string = "Test files"
	file string = "Les panthères modernistes"
)

var File = filepath.Join(Dir, file)

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
