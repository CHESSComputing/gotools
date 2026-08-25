package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/vkuznet/cryptoutils"
)

func decrypt(salt, cipher, entry, fin string) {
	if fin != "" {
		bindata, err := os.ReadFile(fin)
		if err != nil {
			fmt.Printf("fail to open output file %s, error=%v", fin, err)
			os.Exit(1)
		}
		data, err := cryptoutils.Decrypt(bindata, salt, cipher)
		if err != nil {
			fmt.Printf("failt to decrypt %v", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}
	if entry == "" {
		fmt.Println("ERROR: no entry data to encrypt is provided")
		os.Exit(1)
	}
	src := []byte(entry)
	dst := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(dst, src)
	if err != nil {
		fmt.Printf("fail to decrypt %v", err)
		os.Exit(1)
	}
	data, err := cryptoutils.Decrypt(dst[:n], salt, cipher)
	if err != nil {
		fmt.Printf("failt to decrypt %v", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func encrypt(salt, cipher, entry, fout string) {
	if entry == "" {
		fmt.Println("ERROR: no entry data to encrypt is provided")
		os.Exit(1)
	}
	data, err := cryptoutils.Encrypt([]byte(entry), salt, cipher)
	if err != nil {
		fmt.Printf("failt to encrypt %v", err)
		os.Exit(1)
	}
	if fout != "" {
		file, err := os.Create(fout)
		if err != nil {
			fmt.Printf("fail to open output file %s, error=%v", fout, err)
			os.Exit(1)
		}
		defer file.Close()
		file.Write(data)
		return
	}

	dst := make([]byte, hex.EncodedLen(len(data)))
	hex.Encode(dst, data)
	fmt.Println(string(dst))
}

func main() {
	var action string
	flag.StringVar(&action, "action", "", "action to use <encrypt|decrypt>")
	var secret string
	flag.StringVar(&secret, "secret", "", "secret to use for encryption")
	var entry string
	flag.StringVar(&entry, "entry", "", "data to encrypt")
	var cipher string
	flag.StringVar(&cipher, "cipher", "aes", "cipher to use, default aes (aes and nacl are supported)")
	var fin string
	flag.StringVar(&fin, "fin", "", "input file")
	var fout string
	flag.StringVar(&fout, "fout", "", "output file")
	flag.Parse()
	if action == "" {
		fmt.Println("ERROR: no action is provided, please specify either encrypt or decrypt")
		os.Exit(1)
	}
	if secret == "" {
		fmt.Println("ERROR: no secret is provided")
		os.Exit(1)
	}
	if action == "encrypt" {
		encrypt(secret, cipher, entry, fout)
	} else if action == "decrypt" {
		decrypt(secret, cipher, entry, fin)
	} else {
		fmt.Println("ERROR: unsupported action")
		os.Exit(1)
	}
}
