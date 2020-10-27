package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var g_path = "/mnt/hgfs/CLASS_DATA/NLT/yelp/"
var g_reviews = g_path + "reviews_short.json"
var g_verbosity uint = 4

type yelpReview struct {
	Review_id   string  `json:"review_id"`   // string, 22 character unique review id
	User_id     string  `json:"user_id"`     // string, 22 character unique user id, maps to the user in user.json
	Business_id string  `json:"business_id"` // string, 22 character business id, maps to business in business.json
	Stars       float32 `json:"stars"`       // integer, star rating
	Date        string  `json:"date"`        // string, date formatted YYYY-MM-DD
	Text        string  `json:"text"`        // string, the review itself
	Useful      int     `json:"useful"`      // integer, number of useful votes received
	Funny       int     `json:"funny"`       // integer, number of funny votes received
	Cool        int     `json:"cool"`        // integer, number of cool votes received
}

func main() {
	var data []byte

	ifPrintln(2, "About to read file: "+g_reviews)

	// Read input file name
	if len(os.Args) > 1 {
		g_reviews = os.Args[1]
		fmt.Println("Filename passed on command line: " + g_reviews)
	} else {
		fmt.Println("Using default filename: " + g_reviews)
	}
	data = readDataFromFile(g_reviews)

	dec := json.NewDecoder(bytes.NewReader(data))

	for {
		var m yelpReview
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("==========================\n%s: %s\n", m.Review_id, m.Text)
	}

}

func readDataFromFile(fn string) []byte {
	ifPrintln(3, "readDataFromFile(\""+fn+"\"): ")
	defer ifPrintln(3, "readDataFromFile complete.")

	dataFile, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Fatal("ERROR: opening data file (%s). ", err.Error())
	}

	if strings.HasSuffix(fn, ".gz") { // Compressed file
		ifPrintln(3, "Reading compressed file...")
		defer ifPrintln(3, "Decompression complete.")

		zr, err := gzip.NewReader(bytes.NewReader(dataFile))
		if err != nil {
			log.Fatal("ERROR: reading compressed (%s). ", err.Error())
		}
		if dataFile, err = ioutil.ReadAll(zr); err != nil {
			log.Fatal(err)
		}
		zr.Close()
	}
	return dataFile
}

// Prints an error message if verbosity level is less than g_verbosity threshold
func ifPrintln(level int, msg string) {
	if level > 0 { // stderr (level<0) is exempt from quiet
		return
	}
	if uint(math.Abs(float64(level))) <= g_verbosity {
		if level < 0 {
			fmt.Fprintf(os.Stderr, msg+"\n")
		} else {
			fmt.Fprintf(os.Stdout, msg+"\n")
		}
	}
}
