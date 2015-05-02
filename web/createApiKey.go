package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"time"
)

const apiKeyFile = "apikey.txt"

func main() {
	t := fmt.Sprintf("%d", time.Now().Unix())

	h := sha1.New()
	h.Write([]byte(t))
	key := h.Sum(nil)

	if err := ioutil.WriteFile(apiKeyFile, []byte(fmt.Sprintf("%x", key)), 0600); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Api key created and written to file: " + apiKeyFile)
	}
}
