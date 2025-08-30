package main
// نمونه‌های آماده از Google News (کلیدواژه‌های Base)
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
log.Println("چیزی برای پست کردن در این بازه زمانی پیدا نشد")
return
}


sort.Slice(all, func(i, j int) bool { return all[i].Published.After(all[j].Published) })
if len(all) > maxPosts {
all = all[:maxPosts]
}


for _, it := range all {
// پیام کوتاه و تمیز
msg := fmt.Sprintf("🟦 BASE | %s\n%s", it.Title, it.Link)
if len(msg) > 3900 { // حاشیه امن زیر 4096 کاراکتر
msg = msg[:3890] + "…"
}
if err := sendToTelegram(token, chatID, msg); err != nil {
log.Printf("[send err] %v", err)
continue
}
log.Printf("sent: %s", it.Title)
}
}
