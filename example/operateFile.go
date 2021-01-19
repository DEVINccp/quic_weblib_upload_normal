package main

import (
	"io"
	"log"
	"os"
	"strconv"
)

func MergeFile(uuid string, fileLength int) {
	fragmentCount := (fileLength-1)/FRAGMENTSIZE + 1
	os.Chdir(FILEPATH)
	file, err := os.OpenFile(uuid+"_"+strconv.Itoa(1), os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		if err != io.EOF {
			log.Fatal(err)
		}
	}

	bytes := make([]byte, FRAGMENTSIZE)
	for i := 2; i <= fragmentCount; i++ {
		os.Chdir(FILEPATH)
		indexPath := uuid+"_"+strconv.Itoa(i)
		openFile, err := os.OpenFile(indexPath, os.O_RDONLY, 0666)
		if err != nil {
			if err != io.EOF {
				panic("Open file failed")
			}
		}
		read, err := openFile.Read(bytes)
		if err != nil {
			if err != io.EOF {
				panic("Read file failed")
			}
		}
		file.Write(bytes[:read])
		openFile.Close()
		os.Remove(indexPath)
	}
	file.Close()
	err = os.Rename(uuid+"_1", uuid)
	if err != nil {
		panic(err)
		return
	}
}
