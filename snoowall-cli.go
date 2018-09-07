package main

/*
	1. Imgur Integration (Done)
	2. Wallpaper Setting (Done)
	3. Debugging (Done)
	4. Optimization: Switch to indexed Lurker implementation (Done)
						  Change file name to Name (Done)
						  Bring binary data files (Done)
	5. Command Line Options (Done)
	6. Logging (Done) -> Improvement
	7. Configuration File
	8. PNG Problem (Done)
	9. NSFW (Done)
	10. Develop Syncing (Done)
	11. Auto Syncing (Done)
	12. Sync Randomizer
	. 	 Selective Harvest?
	.
	.

	NaN. Graphical User Interface
*/

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/reujab/wallpaper"
	flag "github.com/spf13/pflag"
	"github.com/turnage/graw/reddit"
)

var subreddit string
var top, nsfw, sync bool
var logfile = "LOG.log"
var index int

var rcount = 0

func saveWall(filename string, b []byte) error {
	err := ioutil.WriteFile(filename, b, 0600)
	return err
}

func setWall(file string) error {
	// background, err := wallpaper.Get()
	// if err != nil {
	// 	fmt.Println("[INFO] Can't find previous wallpaper:", err)
	// }

	err := wallpaper.SetFromFile(file)
	if err == nil {
		fmt.Println("Updated Wallpaper:", file)
		log.Println("[INFO] Updated wallpaper:", file)
	}
	return err
}

type saveData struct {
	Time      time.Time
	Subreddit string
	Info      []postMeta
}
type postMeta struct {
	Title     string
	ID        string
	Permalink string
	NSFW      bool
	URL       string
}

/*
	The main code
*/
func main() {
	// collect flags
	flags := flag.NewFlagSet("snoowall-cli", flag.ExitOnError)
	flags.BoolVarP(&top, "top", "t", false, "Select the top wallpaper instead of a random one.")
	flags.BoolVarP(&nsfw, "allow-nsfw", "n", false, "Gives a pass to NSFW content that is blocked by default.")
	flags.BoolVarP(&sync, "sync", "S", false, "Syncs the local database with reddit.")
	flags.IntVar(&index, "index", 1, "nonsense")

	flags.MarkHidden("top")
	flags.MarkHidden("index")
	flags.Parse(os.Args[1:])
	subreddit = flags.Args()[0]

	// setup logging
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("[ERROR] Error opening logfile:", err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	fmt.Printf("[DEBUG] Arguments: subreddit=%s;  top=%t;  index=%d;  sync=%t;  allow-nsfw=%t;  tail:%v\n", subreddit, top, index, sync, nsfw, flags.Args())

	// initialize script to API
	rate := 5 * time.Second
	script, err := reddit.NewScript("graw:snoowall:0.3.1 by /u/psychemerchant", rate)
	if err != nil {
		log.Fatalln("[FATAL] Failed to create script handle: ", err)
		return
	}

	// if cache does not exist, sync
	syncLoc := fmt.Sprintf("%s/%s", os.Getenv("HOME"), ".cache/snoowall-cli")
	cacheloc := fmt.Sprintf("%s/%s", syncLoc, subreddit)
	if _, err := os.Stat(cacheloc); os.IsNotExist(err) {
		sync = true
	}

	// generating cache
	if sync == true {
		if _, err := os.Stat(syncLoc); os.IsNotExist(err) {
			os.MkdirAll(syncLoc, os.ModePerm)
		}
		fmt.Printf("[INFO] Syncing... /r/%s to %s\n", subreddit, cacheloc)

		harvest, err := script.Listing(fmt.Sprintf("/r/%s", subreddit), "")
		if err != nil {
			log.Fatalf("[FATAL] Failed to fetch /r/%s: %s", subreddit, err)
		}
		var subdata saveData
		subdata.Time = time.Now()
		subdata.Subreddit = subreddit
		subdata.Info = make([]postMeta, 0)

		length := len(harvest.Posts)
		if length == 0 {
			log.Fatalln("[ERROR]: No posts! Subreddit might not exist.")
		}
		for i := 0; i < length; i++ {
			post := harvest.Posts[i]
			subdata.Info = append(subdata.Info, postMeta{post.Title, post.ID, post.Permalink, post.NSFW, post.URL})
		}

		var buff bytes.Buffer
		enc := gob.NewEncoder(&buff)
		err = enc.Encode(subdata)
		if err != nil {
			log.Fatalln("[FATAL]: Encoding error", err)
		}
		ioutil.WriteFile(cacheloc, buff.Bytes(), 0600)
		log.Printf("[INFO] Synced /r/%s to %s", subreddit, cacheloc)
	}

	data, err := ioutil.ReadFile(fmt.Sprintf("%s", cacheloc))
	if err != nil {
		log.Fatalln("[ERROR]: Cache file read error.")
	}
	dec := gob.NewDecoder(bytes.NewReader(data))
	var cursubdata saveData
	cursubdata.Info = make([]postMeta, 0)
	err = dec.Decode(&cursubdata)
	var post postMeta
retry:
	if top == true {
		post = cursubdata.Info[index]
	} else if top == false {
		rand.Seed(time.Now().UTC().UnixNano())
		post = cursubdata.Info[rand.Intn(len(cursubdata.Info))]
	}

	// if allow-nsfw - false, nsfw check, retry
	if nsfw == false {
		if post.NSFW == true {
			if rcount == 3 {
				fmt.Println("[DEBUG] Post is NSFW. Try an SFW subreddit.")
				return
			}
			if top == false {
				fmt.Println("[DEBUG] Post is NSFW. Retrying...")
				rcount++
				goto retry
			} else {
				fmt.Println("[DEBUG] Top post is NSFW.")
				return
			}
		}
	}
	fmt.Printf("Title: %s\nURL: %s\n", post.Title, post.URL)
	resp, err := http.Get(post.URL)
	filetype := post.URL[len(post.URL)-4:]
	if filetype != ".jpg" && filetype != ".png" {
		log.Println("[ERROR] Not an image.")
		fmt.Println("[ERROR] Not an image.")
		goto retry
	}
	if err != nil {
		log.Println("[ERROR]: Couldn't fetch resource:", post.URL, err)
		fmt.Println("[ERROR]: Couldn't fetch resource:", post.URL, err)
		return
	}
	body, _ := ioutil.ReadAll(resp.Body)

	// if save=true, save in user's home directory, else save in cache
	var loc string
	loc = fmt.Sprintf("%s/%s", os.Getenv("HOME"), ".cache/snoowall-cli/")
	filename := fmt.Sprintf("%s%s_%s%s", loc, subreddit, post.ID, filetype)
	err = saveWall(filename, body)
	if err != nil {
		log.Fatalln("[ERROR] Wallpaper saving error:", err)
	}

	err = setWall(filename)
	if err != nil {
		log.Println("[ERROR] Wallpaper setting error:", err)
		return
	}
}
