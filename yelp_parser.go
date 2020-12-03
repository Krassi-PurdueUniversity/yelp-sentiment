/*
* CNIT 58100-NLT - course project
* Includes from from https://github.com/cdipaolo/sentiment
 */

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

	"github.com/cdipaolo/sentiment"
	_ "github.com/cdipaolo/sentiment"
	_ "github.com/go-sql-driver/mysql"
)

var g_path = "~/work/yelp"
var g_reviews = g_path + "reviews_short.json"
var g_outputDir = "~/work/yelp/output"
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
	//	ifPrintln(2, "About to read file: "+g_reviews)

	// Read input file name
	if len(os.Args) > 1 {
		if os.Args[1] == "." {
			fmt.Println("Using default filename: " + g_reviews)
		} else {
			g_reviews = os.Args[1]
		}
		fmt.Println("Filename passed on command line: " + g_reviews)
	} else {
		log.Fatal("Please, use: " + os.Args[0] + " <datafile> <action>")
	}

	//fmt.Printf("ARGN: %d \n", len(os.Args))
	if len(os.Args) < 3 {
		log.Fatal("Missing action command. Try: print")
	}

	yelpData := readReviews(g_reviews)

	switch os.Args[2] {
	case "print":
		for k, m := range yelpData {
			fmt.Printf("==========================\nK: %s => %s: %s\n", k, m.Review_id, m.Text)
		}
		break
	case "split":
		counter := 0
		for _, m := range yelpData {
			//fmt.Printf("==========================\nK: %s => %s: %s\n", k, m.Review_id, m.Text)
			fn := fmt.Sprintf("%s/%6.6d_%d.txt", g_outputDir, counter, int(m.Stars))
			f, err := os.Create(fn)
			check(err)
			n, err := f.Write([]byte(m.Text))
			check(err)
			fmt.Printf("file: %s => bytes: %d \n", fn, n)
			f.Close()
			counter++
		}
		break
	case "oldsentiment":
		oldsentiment(yelpData)
	default:
		log.Fatalf("Unknown option \"%s\".", os.Args[2])
	}

}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func oldsentiment(data map[string]yelpReview) {
	model, err := sentiment.Restore()
	if err != nil {
		panic(fmt.Sprintf("Could not restore model!\n\t%v\n", err))
	}

	var positive_guess, positive_fail, negative_guess, negative_fail int
	for k, m := range data {
		an := model.SentimentAnalysis(m.Text, sentiment.English)
		fmt.Printf("==========================\nK: %s (%2.2f): %d\n %s\n", k, m.Stars, an.Score, m.Text)
		fmt.Printf("=====\nSentences: %v\n", an.Sentences)
		fmt.Printf("=====\nWords: %v \n", an.Words)

		if m.Stars > 3 {
			if an.Score == 1 {
				positive_guess++
			} else {
				positive_fail++
			}
		} else {
			if an.Score == 0 {
				negative_guess++
			} else {
				negative_fail++
			}

		} // Ignore 3 stars
	}
	var sum float64
	sum = float64(positive_guess + positive_fail + negative_guess + negative_fail)
	fmt.Printf("S:==============================================\nS:* 3 star reviews are ignorred\n"+
		"S:true positive: %d\nS:true negative: %d\nS:false positive: %d (%f)\nS:false negative: %d (%f)\n",
		positive_guess, negative_guess, positive_fail, float64(positive_fail)/sum, negative_fail, float64(negative_fail)/sum)
}

func readReviews(fn string) map[string]yelpReview {
	//var data []byte
	results := make(map[string]yelpReview)

	data := readDataFromFile(fn)
	dec := json.NewDecoder(bytes.NewReader(data))

	for {
		var m yelpReview
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		//fmt.Printf("==========================\n%s: %s\n", m.Review_id, m.Text)
		results[m.Review_id] = m
	}
	return results
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
