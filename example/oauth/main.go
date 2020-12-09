package oauth

import (
	"fmt"
	jose "github.com/dvsekhvalnov/jose2go"
	Rsa "github.com/dvsekhvalnov/jose2go/keys/rsa"
	"io/ioutil"
)

func DecodeRsaToken(accessToken string) bool{
	keyBytes, err := ioutil.ReadFile("public.key")
	if err != nil {
		panic("invalid key file")
	}

	publicKey, er := Rsa.ReadPublic(keyBytes)
	if er != nil {
		panic("invalid key format")
	}

	payload, headers, err := jose.Decode(accessToken, publicKey)
	if err != nil {
		panic("invalid access token")
	}

	fmt.Printf("\npayload = %v \n", payload)
	fmt.Printf("\nheaders = %v \n", headers)
	return true
}
