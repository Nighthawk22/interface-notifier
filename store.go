package main

import (
	"encoding/gob"
	"os"
)

//Save Encode via Gob to file
func Save(path string, object interface{}) error {
	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		return err
	}
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(object)
	if err != nil {
		return err
	}
	return nil
}

//Load Decode Gob file
func Load(path string, object interface{}) error {
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		return err
	}

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(object)
	if err != nil {
		return err
	}

	return nil
}
