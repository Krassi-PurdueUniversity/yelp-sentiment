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

var g_path = ""
var g_reviews = g_path + "reviews_short.json"
var g_outputDir = "output"
var g_outputModel = "yelp_model.json"
var g_outputPrefix = ""
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
	var command, arg string

	if len(os.Args) > 2 {
		command = os.Args[1]
		arg = os.Args[2]

		//fmt.Printf("DEBUG: %s :: %s \n", command, arg)
		// Sets default name for the reviews file - this is not used in all cases
		if arg == "." {
			arg = g_reviews
		}

		//fmt.Printf("DEBUG: %s :: %s \n", command, g_reviews)
	} else {
		log.Fatalf("Usage %s <command> <argument>", os.Args[0])
	}

	switch command {
	case "print":

		yelpData := readReviews(arg)

		for k, m := range yelpData {
			fmt.Printf("==========================\nK: %s => %s: %s\n", k, m.Review_id, m.Text)
		}
		break
	case "split":
		counter := 0
		g_outputPrefix = os.Args[3]

		yelpData := readReviews(arg)
		for _, m := range yelpData {
			//fmt.Printf("==========================\nK: %s => %s: %s\n", k, m.Review_id, m.Text)
			fn := fmt.Sprintf("%s/%s-%6.6d_%d.txt", g_outputDir, g_outputPrefix, counter, int(m.Stars))
			f, err := os.Create(fn)
			check(err)
			n, err := f.Write([]byte(m.Text))
			check(err)
			fmt.Printf("file: %s => bytes: %d \n", fn, n)
			f.Close()
			counter++
		}
		break
	case "train":
		var err error
		//var myModel sentiment.Models
		model, err := sentiment.Train()
		if err != nil {
			panic(err.Error())
		}

		sentiment.PersistToFile(model, g_outputModel)

		an := model.SentimentAnalysis("I feel good!", sentiment.English)
		fmt.Printf("==========================\nScore: %d\n", an.Score)
		fmt.Printf("=====\nSentences: %v\n", an.Sentences)
		fmt.Printf("=====\nWords: %v \n", an.Words)

		break
	case "ratedir":
		rateDir(arg)

		break
	case "oldsentiment":
		yelpData := readReviews(arg)
		oldsentiment(yelpData)
	default:
		log.Fatalf("Unknown option \"%s\".", command)
	}

}

func rateDir(dir string) {
	var pos, neg int32

	fmt.Printf("Rating directory: %s\n", dir)
	files, err := ioutil.ReadDir(dir)
	check(err)

	model, err := sentiment.Restore()

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".txt") { // Ignore files not ending with .txt
			continue
		}

		fmt.Printf("Processing: %s => ", f.Name())
		content, err := ioutil.ReadFile(dir + f.Name())
		check(err)
		//text := string(content)
		an := model.SentimentAnalysis(string(content), sentiment.English)
		fmt.Printf("%d\n", an.Score)
		//fmt.Printf("%s :: %s \n", an.Sentences, an.Words)
		if an.Score == 1 {
			pos++
		} else {
			neg++
		}
	}
	fmt.Printf("Analyzed total of %d files. Pos: %d; Neg: %d .\n", pos+neg, pos, neg)
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
		log.Fatalf("ERROR: opening data file (%s): %s. ", fn, err.Error())
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
