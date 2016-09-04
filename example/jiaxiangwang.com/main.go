package main

import (
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/miaolz123/conver"
	"github.com/miaolz123/crawler"
)

type event struct {
	Province    string
	City        string
	County      string
	Title       string
	Description string
	Category    string
}

func main() {
	option := crawler.Option{
		Name:      "家乡网",
		PauseTime: []int{300, 1000},
	}
	c := crawler.New(option)
	c.AddRule("default", crawler.Rule{
		Parse: func(ctx *crawler.Context) bool {
			if ctx.Response.StatusCode > 200 {
				log.Println("网络访问错误：", ctx.Response.StatusCode)
				time.Sleep(time.Minute)
				return false
			}
			provinces := ctx.Document.Find("li.region a")
			log.Printf("解析到省份的数量：%v\n", provinces.Length())
			provinces.Each(func(i int, s *goquery.Selection) {
				url, ok := s.Attr("href")
				url = strings.TrimPrefix(url, "cn/")
				if ok && url != "" {
					c.AddQueue(crawler.Queue{
						URL:   "http://www.jiaxiangwang.com/cn/" + url,
						Rule:  "province",
						Param: map[string]interface{}{"province": s.Text()},
					})
				}
			})
			return true
		},
	})
	c.AddRule("province", crawler.Rule{
		Parse: func(ctx *crawler.Context) bool {
			if ctx.Response.StatusCode > 200 {
				log.Println("网络访问错误：", ctx.Response.StatusCode)
				time.Sleep(time.Minute)
				return false
			}
			province := conver.StringMust(ctx.Param["province"])
			log.Printf("开始解析省份：%v\n", province)
			categorys := ctx.Document.Find("li.category a")
			categorys.Each(func(i int, s *goquery.Selection) {
				url, ok := s.Attr("href")
				url = strings.TrimPrefix(url, "cn/")
				if ok && url != "" {
					category := strings.TrimSpace(s.Text())
					c.AddQueue(crawler.Queue{
						URL:   "http://www.jiaxiangwang.com/cn/" + url,
						Rule:  "category",
						Param: map[string]interface{}{"province": province, "category": category},
					})
				}
			})
			return true
		},
	})
	c.AddRule("category", crawler.Rule{
		Parse: func(ctx *crawler.Context) bool {
			if ctx.Response.StatusCode > 200 {
				log.Println("网络访问错误：", ctx.Response.StatusCode)
				time.Sleep(time.Minute)
				return false
			}
			province := conver.StringMust(ctx.Param["province"])
			category := conver.StringMust(ctx.Param["category"])
			log.Printf("开始解析%v 的 %v\n", province, category)
			article := ctx.Document.Find("article.main").Children()
			event := event{
				Province: province,
				Category: category,
			}
			article.Each(func(i int, s *goquery.Selection) {
				switch goquery.NodeName(s) {
				case "h2":
					if !strings.Contains(s.Text(), "跨地区") {
						event.City = strings.TrimSpace(s.Text())
					}
				case "h5":
					if !strings.Contains(s.Text(), "跨地区") {
						event.County = strings.TrimSpace(s.Text())
					}
				case "dl":
					s.Children().Each(func(j int, s *goquery.Selection) {
						switch goquery.NodeName(s) {
						case "dt":
							if event.Title != "" {
								c.AddDataToStorer("event", event)
							}
							event.Title = strings.TrimSpace(s.Text())
						case "dd":
							event.Description = strings.TrimSpace(s.Text())
							c.AddDataToStorer("event", event)
							event.Title = ""
							event.Description = ""
						}
					})
				}
			})
			return true
		},
	})
	c.AddQueue(crawler.Queue{
		URL:  "http://www.jiaxiangwang.com/",
		Rule: "default",
	})
	c.Run()
}
