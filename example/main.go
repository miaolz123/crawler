package main

import (
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/miaolz123/conver"
	"github.com/miaolz123/crawler"
)

type repository struct {
	Name        string
	Owner       string
	URL         string
	Language    string
	Description string
	Watch       int
	Star        int
	Fork        int
}

func main() {
	c := crawler.New(crawler.Option{
		Name: "Github Trending",
	})
	c.AddQueue(crawler.Queue{
		URL: "https://github.com/trending",
	})
	c.AddRule("default", crawler.Rule{
		Parse: func(ctx *crawler.Context) bool {
			if ctx.Response.StatusCode > 200 {
				log.Println("HTTP error code:", ctx.Response.StatusCode)
				return false
			}
			repoList := ctx.Document.Find("li.repo-list-item")
			repoList.Each(func(i int, s *goquery.Selection) {
				repo := repository{}
				repo.Owner = s.Find("span.prefix").Text()
				repo.Name = strings.TrimPrefix(strings.TrimSpace(s.Find("h3.repo-list-name").Text()), repo.Owner)
				repo.Name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(repo.Name), "/"))
				repo.Description = strings.TrimSpace(s.Find("p.repo-list-description").Text())
				metas := strings.Split(s.Find("p.repo-list-meta").Text(), "•")
				if len(metas) == 3 {
					repo.Language = strings.TrimSpace(metas[0])
				}
				url, _ := s.Find("h3.repo-list-name a").Attr("href")
				if url != "" {
					repo.URL = "https://github.com" + url
					c.AddQueue(crawler.Queue{
						URL:   repo.URL,
						Rule:  "repository",
						Param: map[string]interface{}{"repo": repo},
					})
				}
			})
			return true
		},
	})
	c.AddRule("repository", crawler.Rule{
		Parse: func(ctx *crawler.Context) bool {
			if ctx.Response.StatusCode > 200 {
				log.Println("HTTP error code:", ctx.Response.StatusCode)
				return false
			}
			repo := ctx.Param["repo"].(repository)
			counts := ctx.Document.Find("a.social-count")
			if counts.Length() < 3 {
				return false
			}
			repo.Watch = conver.IntMust(counts.First().Text())
			repo.Star = conver.IntMust(counts.Slice(1, 2).Text())
			repo.Fork = conver.IntMust(counts.Last().Text())
			c.AddDataToStorer("repo", repo)
			return true
		},
	})
	c.Run()
}
