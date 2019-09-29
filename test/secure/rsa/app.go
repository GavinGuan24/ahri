package main

import (
	"fmt"
	"github.com/GavinGuan24/ahri/core"
)

var publicKey = []byte(`
-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDjoqIcnz/v1uu6cxrp2pD561hO
zdV7dLrWYQjUpyGME/DMzu+JzkKbjCJZXut4yQITM1qv9L8IryqRkUlotFTEP1UK
iPOjLYhkhRYPQoJKnAQgIefl4kqB/RC9cursYY+xaxjVO1ijH+aX4Al5Q5jcCoJi
0kc3BJ6fwFpzMY2JoQIDAQAB
-----END PUBLIC KEY-----
`)

var privateKey = []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQDjoqIcnz/v1uu6cxrp2pD561hOzdV7dLrWYQjUpyGME/DMzu+J
zkKbjCJZXut4yQITM1qv9L8IryqRkUlotFTEP1UKiPOjLYhkhRYPQoJKnAQgIefl
4kqB/RC9cursYY+xaxjVO1ijH+aX4Al5Q5jcCoJi0kc3BJ6fwFpzMY2JoQIDAQAB
AoGAZr4lBV4rcYlD5GfHof1wqhy7QvZMgOhy3Af4AGNfFOZ7LTXJkB10mthpOIVL
Kr0vHpNzPy/seXL2d7VnuMaL6x2e7Y3t/iwu/rVeqItQp1qsTuatKHV15HLu77MH
6WkDGropZvj24O1LiYanOvO5zZMRVJnr41xl02lNx07TaTECQQD+zXEkdsskEnql
nYEIBW8tDKxjf+1LyC80v97kgk+B/Hnb3tsQsEXjH+2TiJyy7T4yqbEDjRMeE8wC
/uBllZUvAkEA5LSBf6qtXmctB1bJVsKfuSfnV447PVATfmCQKgUVDxniHIBYg8hw
T3c6JviOgrZ5scayMd5opj8VqOw8l726LwJAMvZ8TsLD1q8rgLyD9kq/9c63HB+W
IrYjWvWVazb1GBabePKV9jyLfeYA6qVEUjVJX3C5SvCIhleHUoIP98F3WQJAOKmU
D/pMW8A6QsA4v9sWUXxWb7XYbXdibQQlk5OQxR4HjEIsK/JECRwj9zXLsQzel7H/
wiU1TkMA7cohtQKXlwJBAO753O4xOL5LgmI2CAkrTNqv0v0KFawLnMTxqQ6lCz43
0eAKXFLsb0y+Au5tIDpWpXMQuHGzMRs74AVLUj/uLro=
-----END RSA PRIVATE KEY-----
`)

/*

#!/usr/bin/env bash

openssl genrsa -out rsa_private_key.pem 1024
openssl rsa -in rsa_private_key.pem -pubout -out rsa_public_key.pem

*/

func main() {
	password := "I am Groot. I am Groot. I am Groot. I am Groot. I am Groot. I am Groot. I am Groot. I am Groot. "
	originData := []byte(password)

	//your can
	pubkey := publicKey
	prikey := privateKey

	ciphertext, e := core.EncryptRsa(originData, pubkey)
	if e != nil {
		fmt.Println(e)
		return
	}
	data, e := core.DecryptRsa(ciphertext, prikey)
	if e != nil {
		fmt.Println(e)
		return
	}

	fmt.Println(string(data) == string(originData))
}
