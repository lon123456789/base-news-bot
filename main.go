package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type Item struct {
	Title     string
	Link      string
	Published time.Time
	Source    string
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func fetchFeed(fp *gofeed.Parser, feedURL string, window time.Duration) ([]Item, error) {
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return nil, err
	}
	items := []Item{}
	cutoff := time.Now().Add(-window)
	for _, it := range feed.Items {
		if it.PublishedParsed == nil {
			continue
		}
		pub := it.PublishedParsed.UTC()
		if pub.Before(cutoff) {
			continue
		}
		items = append(items, Item{
			Title:     strings.TrimSpace(it.Title),
			Link:      strings.TrimSpace(it.Link),
			Published: pub,
			Source:    feed.Title,
		})
	}
	return items, nil
}

func sendToTelegram(token, chatID, text string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	values := url.Values{}
	values.Set("chat_id", chatID)
	values.Set("text", text)
	values.Set("disable_web_page_preview", "true")

	resp, err := http.PostForm(endpoint, values)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API status: %s", resp.Status)
	}
	return nil
}

func main() {
	// environment variables
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHANNEL") // like @my_base_news
	if token == "" || chatID == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN and TELEGRAM_CHANNEL must be set")
	}

	windowHours := getenv("WINDOW_HOURS", "12")
	wh, err := time.ParseDuration(windowHours + "h")
	if err != nil {
		wh = 12 * time.Hour
	}

	maxPosts := 10
	if mp := getenv("MAX_POSTS", ""); mp != "" {
		var m int
		if err := json.Unmarshal([]byte(mp), &m); err == nil && m > 0 {
			maxPosts = m
		}
	}

	feedsEnv := os.Getenv("FEEDS")
	feeds := []string{}
	if feedsEnv != "" {
		for _, f := range strings.Split(feedsEnv, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				feeds = append(feeds, f)
			}
		}
	} else {
		feeds = []string{
			"https://news.google.com/rss/search?q=Base%20network%20crypto%20OR%20Base%20L2%20OR%20Coinbase%20Base&hl=en-US&gl=US&ceid=US:en",
			"https://news.google.com/rss/search?q=site%3Abase.org&hl=en-US&gl=US&ceid=US:en",
		}
	}

	fp := gofeed.NewParser()
	all := []Item{}
	seen := map[string]bool{}

	for _, u := range feeds {
		items, err := fetchFeed(fp, u, wh)
		if err != nil {
			log.Printf("[warn] feed error %s: %v", u, err)
			continue
		}
		for _, it := range items {
			key := strings.TrimSpace(it.Link)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, it)
		}
	}

	if len(all) == 0 {
		log.Println("No new posts found in this window")
		return
	}

	sort.Slice(all, func(i, j int) bool { return all[i].Published.After(all[j].Published) })
	if len(all) > maxPosts {
		all = all[:maxPosts]
	}

	for _, it := range all {
		msg := fmt.Sprintf("🟦 BASE | %s\n%s", it.Title, it.Link)
		if len(msg) > 3900 {
			msg = msg[:3890] + "…"
		}
		if err := sendToTelegram(token, chatID, msg); err != nil {
			log.Printf("[send err] %v", err)
			continue
		}
		log.Printf("sent: %s", it.Title)
	}
}
