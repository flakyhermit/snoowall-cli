package main

/*
	1. Imgur Integration (Done)
	2. Wallpaper Setting (Done)
	3. Debugging (Done)
	4. Optimization: Switch to indexed Lurker implementation, Change file name to Name (Done), Bring binary data files
	5. Command Line Options (Done)
	6. Logging
	7. Configuration File
	8. PNG Problem (Done)
	9. NSFW
	.
	.
	NaN. Graphical User Interface
*/

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/reujab/wallpaper"
	"github.com/turnage/graw/reddit"
)

var loc = "/home/pro/Dropbox/Code/golang/snoowall/Wallpapers/"
var datafile = "data"
var name = "info.agent"
var path = fmt.Sprintf("%s%s", loc, name)
var index int
var subreddit string
var top, nsfw bool

func saveWall(filename string, b []byte) error {
	err := ioutil.WriteFile(filename, b, 0600)
	return err
}

func setWall(file string) error {
	background, err := wallpaper.Get()
	if err != nil {
		fmt.Println("[DEBUG] Can't find previous wallpaper:", err)
	}
	fmt.Println("Current wallpaper:", background)
	err = wallpaper.SetFromFile(file)
	if err == nil {
		fmt.Println("Updated Wallpaper:", file)
	}
	return err
}

/*
	The main code
*/
func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	flag.StringVar(&subreddit, "sub", "wallpaper", "Specify the subreddit to fetch wallpapers from.")
	flag.BoolVar(&top, "top", false, "Select the top wallpaper instead of a random one.")
	flag.IntVar(&index, "index", 1, "Post index (0-99)")
	flag.BoolVar(&nsfw, "allow-nsfw", false, "Gives a pass to NSFW content that is blocked by default")
	flag.Parse()
	fmt.Printf("[DEBUG] Arguments: sub:%s;  top:%t;  index:%d;  allow-nsfw:%t;  tail:%v\n", subreddit, top, index, nsfw, flag.Args())

	rate := 5 * time.Second
	script, err := reddit.NewScript("graw:snoowall:0.3.1 by /u/psychemerchant", rate)
	if err != nil {
		fmt.Println("[DEBUG] Failed to create script handle: ", err)
		return
	}

	harvest, err := script.Listing(fmt.Sprintf("/r/%s", subreddit), "")
	if err != nil {
		fmt.Printf("[DEBUG] Failed to fetch /r/%s: %s", subreddit, err)
		return
	}

	var post *reddit.Post
retry:
	if top == true {
		post = harvest.Posts[index]
	} else if top == false {
		post = harvest.Posts[rand.Intn(99)]
	}
	if nsfw == false {
		if post.NSFW == true {
			fmt.Println("[DEBUG] Post is NSFW")
			if top == false {
				goto retry
			}
		}
	}

	fmt.Println("[DEBUG] Post array length: ", len(harvest.Posts))
	// postPermalinks := make([]string, 0, 100)
	// for i := 0; i < 100; i++ {
	// 	post := harvest.Posts[i]
	// 	postPermalinks = append(postPermalinks, post.Permalink)
	// }
	// ioutil.WriteFile("postPermalinks", []byte(fmt.Sprintf("%#v", postPermalinks)), 0600)

	// ioutil.WriteFile(datafile, []byte(post.Name), 0600)
	// fmt.Println("After:", post.Name)
	fmt.Printf("[Title]: %s\n[URL]: %s\n", post.Title, post.URL)

	resp, err := http.Get(post.URL)
	filetype := post.URL[len(post.URL)-4:]
	if filetype != ".jpg" && filetype != ".png" {
		fmt.Println("[DEBUG] Not an image.")
		goto retry
	}
	fmt.Println("[DEBUG] Image Type:", filetype)
	if err != nil {
		fmt.Println("[DEBUG]: Couldn't fetch resource:", post.URL, err)
		return
	}
	body, _ := ioutil.ReadAll(resp.Body)
	filename := fmt.Sprintf("%s%s_%s.%s", loc, subreddit, post.ID, filetype)
	err = saveWall(filename, body)
	if err != nil {
		fmt.Println("[DEBUG] Wallpaper saving error:", err)
	}
	err = setWall(filename)
	if err != nil {
		fmt.Println("[DEBUG] Wallpaper setting error:", err)
		return
	}

}