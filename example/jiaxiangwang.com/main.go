package main

import (
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"github.com/miaolz123/conver"
	"github.com/miaolz123/crawler"
)

const dbConf = "xxxx:XXXX@tcp(192.168.1.1:3306)/xxxxxxx?charset=utf8&loc=Asia%2FShanghai"

type tmpHomelandStuff struct {
	ID          int64 `orm:"auto;pk;column(id)"`
	Province    string
	City        string
	Area        string
	Title       string `orm:"type(text)"`
	Description string `orm:"type(text)"`
	Type        string
}

func init() {
	orm.RegisterModel(new(tmpHomelandStuff))
	err := orm.RegisterDataBase("default", "mysql", dbConf, 30, 30)
	if err != nil {
		log.Panicln(err)
	}
}

func main() {
	option := crawler.Option{
		Name:       "家乡网",
		PauseTime:  []int{300, 1000},
		StorerWork: storerWork,
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
			article := ctx.Document.Find("article.main").Children()
			event := tmpHomelandStuff{
				Province: province,
				Type:     category,
			}
			article.Each(func(i int, s *goquery.Selection) {
				switch goquery.NodeName(s) {
				case "h2":
					if !strings.Contains(s.Text(), "跨地区") {
						event.City = strings.TrimSpace(s.Text())
					}
					event.Area = ""
				case "h5":
					if !strings.Contains(s.Text(), "跨地区") {
						event.Area = strings.TrimSpace(s.Text())
					}
				case "dl":
					skip := 0
					s.Children().Each(func(j int, s *goquery.Selection) {
						if skip > 0 {
							skip--
							return
						}
						switch goquery.NodeName(s) {
						case "dt":
							if event.Title != "" {
								c.AddDataToStorer("event", event)
							}
							event.Title = strings.TrimSpace(s.Text())
						case "dd":
							event.Description = strings.TrimSpace(s.Text())
							for goquery.NodeName(s.Next()) == "dd" {
								skip++
								s = s.Next()
								event.Description += "\n" + strings.TrimSpace(s.Text())
							}
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

func storerWork(storers map[string][]interface{}) map[string]bool {
	db := orm.NewOrm()
	results := make(map[string]bool)
	for name, datas := range storers {
		switch name {
		case "event":
			events := []tmpHomelandStuff{}
			for _, data := range datas {
				events = append(events, data.(tmpHomelandStuff))
			}
			if len(events) > 0 {
				if _, err := db.InsertMulti(30, &events); err != nil {
					log.Println("数据库写入错误：", err)
				} else {
					log.Println("数据库写入成功")
					results[name] = true
				}
			}
		}
	}
	return results
}
