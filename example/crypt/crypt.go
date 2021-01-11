package crypt

import (
	"bytes"
	"crypto/rand"
	"math/big"
)

const (
	IgnoreEncryptFile = ".BMP|.GIF|.JPEG|.SVG|.PNG|.JPG|.AVI|.RMVB|.RM|.ASF|.DIVX|.MPG|.MPEG|.MPE|.WMV|.MP4|.MKV|.VOB|.MP3"
	EncryptFlag = "20ENCRYPTbyWEBLIBsys15Yp"
	EncryptLen = 48
	EncryptKeyLen = 24
	KEY = "10111010"
)

func FileEncrypt(input,output []byte, byteStart int64) []byte {
	randomString := createRandomString()
	_ = append(output, randomString...)
	encryptString := EncryptString()
	keyIndex := byteStart % 32
	for i := 0; i < len(input); i++ {
		_ = append(output,input[i] ^ encryptString[keyIndex])
		keyIndex = keyIndex + 1
		if keyIndex == int64(len(encryptString)) {
			keyIndex = 0
		}
	}
	return output
}

func createRandomString() string {
	var container string
	var str = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	bufferString := bytes.NewBufferString(str).Len()
	newInt := big.NewInt(int64(bufferString))
	for i := 0; i < EncryptLen; i++ {
		if i % 2 == 0 {
			randomInt, _ := rand.Int(rand.Reader, newInt)
			container += string(str[randomInt.Int64()])
		} else {
			container += string(EncryptFlag[i/2])
		}
	}
	return container
}

func EncryptString() []byte {
	return []byte(EncryptFlag + KEY)
}

func FileDecrypt(input *[]byte, byteStart int64){
	encryptString := EncryptString()
	keyIndex := byteStart % 32
	for i := 0; i < len(*input); i++ {
		(*input)[i] = (*input)[i] ^ encryptString[keyIndex]
		keyIndex = keyIndex + 1
		if keyIndex == int64(len(encryptString)) {
			keyIndex = 0
		}
	}
}

