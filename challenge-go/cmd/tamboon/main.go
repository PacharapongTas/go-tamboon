package main

import (
	"bytes"
	"fmt"
	"go-tamboon/cipher"
	"io"
	"os"
)

func main() {
	filepath := "./data/fng.1000.csv.rot128"
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Error when opening file: ", err)
		os.Exit(1)
	}
	defer file.Close()

	readerByRot, err := cipher.NewRot128Reader(file)
	if err != nil {
		fmt.Println("Error creating new RotReader: ", err)
		os.Exit(1)
	}
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, readerByRot)
	if err != nil {
		fmt.Println("Error decrypting file: ", err)
		os.Exit(1)
	}

	// fmt.Println("-- Data After Decode --")
	// fmt.Println(buffer.String())
}
