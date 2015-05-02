// Send test dataset to:
// - host 'h'
// - starting with timestamp 't'
// - each dataset's time interval 'i'
// Example: go run sendTestData.go -h localhost:1664 -t 1345678910 -i 5

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

const apiKeyFile = "apikey.txt"

var apiKey string

func loadApiKey() {
	body, err := ioutil.ReadFile(apiKeyFile)
	if err != nil {
		panic(err)
	}

	apiKey = string(body)
}

func main() {
	hostPtr := flag.String("h", "localhost:1664", "Host")
	tstampPtr := flag.Int64("t", time.Now().Unix(), "Start timestamp")
	intervalPtr := flag.Int64("i", 5, "Interval between each dataset's time (in minutes)")

	flag.Parse()

	fmt.Println("host:", *hostPtr)
	fmt.Println("tstamp:", *tstampPtr)
	fmt.Println("interval:", *intervalPtr)

	startTime := time.Unix(*tstampPtr, 0)

	rand.Seed(time.Now().Unix())
	t := rand.Float32()*5 + 17
	h := rand.Float32()*15 + 55

	loadApiKey()
	ticker := time.NewTicker(time.Millisecond * 5000)

	for range ticker.C {
		t = t + rand.Float32()*2 - 1
		h = h + rand.Float32()*2 - 1

		query := fmt.Sprintf("http://%s/add?apiKey=%s&d=%d&t=%f&h=%f", *hostPtr, apiKey, startTime.Unix(), t, h)
		fmt.Println(query)

		resp, err := http.Get(query)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf("Received status code %d\n", resp.StatusCode)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("%s", body)
			}
		}

		startTime = startTime.Add(time.Minute * time.Duration(*intervalPtr))
	}

}
