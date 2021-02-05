package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alwashali/GoMovie/randomize"

	"github.com/tidwall/gjson"
)

//Get a new key from: https://www.themoviedb.org/documentation/api
// Put your key here
var apikey string = ""
var reader = bufio.NewReader(os.Stdin)

var weightedGenre = map[string]int{
	"action": 1, "adventure": 1,
	"comedy": 1, "documentary": 1,
	"drama": 1, "horror": 1,
	"romance": 1, "war": 1,
	"scifi": 1, "music": 1,
	"history": 1, "family": 1,
	"animation": 1, "crime": 1,
}

type user struct {
	Name   string         `json:"name"`
	Genre  map[string]int `json:"genre"`
	Rating int            `json:"rating"`
}

type movie struct {
	name     string
	year     int
	rating   int
	genre    string
	age      int
	overview string
}

func getGenreID(genre string) int {

	var ID int

	genreIDsURL := "https://api.themoviedb.org/3/genre/movie/list?api_key=4fd5df6c8912b576912303f470560ca5&language=en-US"

	resp, err := http.Get(genreIDsURL)
	if err != nil {
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {

		log.Fatalln(err)
	}
	defer resp.Body.Close()

	result := gjson.Get(string(body), "genres")

	for _, g := range result.Array() {
		gl := strings.ToLower(g.Map()["name"].String())
		if gl == genre {
			// GJSON returns int64
			ID = int(g.Map()["id"].Int())
			return ID
		}
	}
	log.Fatalln("Genre ID Not Found")
	return 0
}

func (u *user) pickMovie() movie {

	mov := movie{}
	rand.Seed(time.Now().UTC().UnixNano())
	genrePicker := randomize.NewChooser()

	//slice to add genres
	genres := make([]randomize.Choice, 0)
	for k, v := range u.Genre {
		if v == 0 {
			continue
		}
		genres = append(genres, randomize.Choice{Item: k, Weight: v})
	}
	genrePicker = randomize.NewChooser(genres...)

	genre := genrePicker.Pick().(string)
	// Get genre ID from themovieDB website
	genreID := getGenreID(genre)

	discoverMoviesURL := "https://api.themoviedb.org/3/discover/movie"
	discoverURL, _ := url.Parse(discoverMoviesURL)
	query, _ := url.ParseQuery(discoverURL.RawQuery)
	query.Add("api_key", apikey)
	query.Add("language", "en-US")
	query.Add("with_genres", strconv.Itoa(genreID))
	query.Add("sort_by", "popularity.desc")
	query.Add("include_adult", "true")
	//query.Add("with_keywords","")
	query.Add("vote_average.gte", strconv.Itoa(u.Rating))
	//query.Add("release_date.gte","2010")
	//query.Add("sort_by","")

	discoverURL.RawQuery = query.Encode()

	resp, err := http.Get(discoverURL.String())
	if err != nil {
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {

		log.Fatalln(err)
	}
	defer resp.Body.Close()

	result := gjson.Get(string(body), "results")
	movies := result.Array()

	x1 := rand.NewSource(time.Now().UnixNano())
	y1 := rand.New(x1)
	if len(movies) < 1 {
		fmt.Println("No Result, Try Again")
		os.Exit(0)
	}
	n := 1 + y1.Intn(len(movies)-1)

	mov.name = movies[n].Map()["title"].String()
	mov.genre = genre
	mov.overview = movies[n].Map()["overview"].String()

	return mov

}

func getUser(name string) (*user, bool) {
	u := &user{}

	jsonFile, err := os.Open(name)
	if err != nil {
		fmt.Println(err)
		return u, false
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)

	json.Unmarshal([]byte(byteValue), &u)

	fmt.Printf("\nAlready Saved\n%s, your preferences: \n", u.Name)
	for k, v := range u.Genre {
		fmt.Printf("%s: %v\n", k, v)
	}

	fmt.Printf("Rating: %d \n", u.Rating)
	return u, true

}

func read(t string) interface{} {
	if t == "int" {
		for {
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			if input == "\n" {
				fmt.Printf("Error, type again\n")
				continue
			}
			input = strings.TrimSuffix(input, "\n")
			num, err := strconv.Atoi(input)
			if err != nil {
				fmt.Println(err)
			}
			return num
		}

	}

	if t == "string" {
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
		}
		input = strings.TrimSuffix(input, "\n")
		return input
	}

	return nil
}

func newUser(name string) *user {
	// weights of each genre is 1
	u := &user{}
	fmt.Println("New Movie Preferences ")
	u.Name = name
	u.Genre = weightedGenre
	fmt.Println("Enter genre preference score out of 10 or 0 to remove genre")
	for k := range u.Genre {
		fmt.Println(k)
		u.Genre[k] = read("int").(int)
	}

	fmt.Println("Movie Rating?, Enter 0 for randome between 6 to 10")
	fmt.Print(">")
	u.Rating = read("int").(int)
	if u.Rating == 0 {
		min := 6
		max := 11
		x1 := rand.NewSource(time.Now().UnixNano())
		y1 := rand.New(x1)
		// generate between 6 and 11 (6 included)
		u.Rating = y1.Intn(max-min) + min
	}
	callClear()
	fmt.Printf("\nGreat %s, your preferences: \n", u.Name)
	for k, v := range u.Genre {
		fmt.Printf("%s: %v\n", k, v)
	}

	fmt.Printf("Rating: %d \n", u.Rating)
	fmt.Print("\n")

	file, err := json.Marshal(u)
	if err != nil || string(file) == "{}" {
		fmt.Println("err")
		fmt.Println("Preferences Not Saved")
		print(string(file))
		return u
	}
	err = ioutil.WriteFile(u.Name, file, 0644)
	if err == nil {
		fmt.Println("Preferences saved ...")
	}
	return u
}

func callClear() {
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok {                          //if we defined a clear func for that platform:
		value() //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
	}
}

var clear map[string]func() //create a map for storing clear funcs

func init() {
	//if no api key
	if apikey == "" {
		fmt.Println("No themoviedb API key found ")
		os.Exit(0)
	}

	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}

	clear["darwin"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func main() {
	nameFlag := flag.String("name", "", "Enter Your Name")

	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Println("-name options is required")
		os.Exit(1)
	}

	if *nameFlag != "" {

		fmt.Printf(`  
		###############################################################################################

		 This tool uses probability to pick a movie based on weights provided for each genre
		 Entre your genre weight from 1 to 10 or 0 to remove movies belonging to a particular genere 

		###############################################################################################
		`)

		user, found := getUser(*nameFlag)
		if found != true {
			callClear()
			user = newUser(*nameFlag)
		}

		fmt.Println("\nPress enter to search for a movie or e to edit preferences")
		input := read("string")
		if input == "e" || input == "E" {
			newUser(*nameFlag)
		}
		callClear()
		time.Sleep(3 * time.Second)
		movie := user.pickMovie()
		fmt.Printf("Title: %s\nGenre: %s\nOverview: %s\n", movie.name, movie.genre, movie.overview)
	}
}
