package crawler

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/emirpasic/gods/lists/arraylist"
)

var (
	// DefaultMethod is "GET"
	DefaultMethod = "GET"
	// DefaultTimeout is 30s
	DefaultTimeout = 30 * time.Second
)

func defaultStorerWork(storers map[string][]interface{}) map[string]bool {
	results := make(map[string]bool)
	for name, datas := range storers {
		for _, data := range datas {
			log.Printf("Get data: %+v", data)
		}
		results[name] = true
	}
	return results
}

// Crawler is just a crawler
type Crawler struct {
	status      int
	option      Option
	rules       map[string]Rule
	queues      arraylist.List
	queuesTemp  arraylist.List
	queuesLock  bool
	storers     map[string][]interface{}
	storersTemp map[string][]interface{}
	storersLock bool
	sync.RWMutex
}

// Option is an option of Crawler
type Option struct {
	Name            string
	LogDisable      bool
	AutoStopDisable bool
	PauseTime       []int
	pauseTimeDif    int
	DefaultMethod   string
	StorerWork      func(map[string][]interface{}) map[string]bool
}

// Rule is a rule for Crawler
type Rule struct {
	name          string
	Timeout       time.Duration
	BeforeRequest func(*http.Request)
	Parse         func(*Context) bool
}

// Queue is a request queue for Crawler
type Queue struct {
	URL    string
	Method string
	Rule   string
	Param  map[string]interface{}
}

// Context is a context for request and response
type Context struct {
	Crawler  *Crawler
	client   http.Client
	Request  *http.Request
	Response *http.Response
	Document *goquery.Document
	Param    map[string]interface{}
}

// New get a new Crawler from option
func New(option Option) *Crawler {
	if option.DefaultMethod == "" {
		option.DefaultMethod = DefaultMethod
	}
	if len(option.PauseTime) < 1 {
		option.PauseTime = []int{1000, 3000}
	} else if len(option.PauseTime) == 1 || option.PauseTime[0]-option.PauseTime[1] > 0 {
		option.PauseTime = []int{option.PauseTime[0], option.PauseTime[0]}
	}
	if option.StorerWork == nil {
		option.StorerWork = defaultStorerWork
	}
	option.pauseTimeDif = option.PauseTime[1] - option.PauseTime[0]
	return &Crawler{
		option:      option,
		rules:       make(map[string]Rule),
		storers:     make(map[string][]interface{}),
		storersTemp: make(map[string][]interface{}),
	}
}

// AddQueue add a queue to Crawler
func (c *Crawler) AddQueue(queue Queue) {
	if queue.URL == "" {
		log.Printf("Crawle[%v] AddQueue() error: Queue.URL is not defined", c.option.Name)
	}
	if queue.Method == "" {
		queue.Method = c.option.DefaultMethod
	}
	if queue.Rule == "" {
		queue.Rule = "default"
	} else {
		queue.Rule = strings.ToLower(queue.Rule)
	}
	if c.queuesLock {
		c.queuesTemp.Add(queue)
	} else {
		c.queues.Add(queue)
	}
}

// AddRule add a rule to Crawler
func (c *Crawler) AddRule(name string, rule Rule) {
	name = strings.ToLower(name)
	rule.name = name
	if rule.Timeout == 0 {
		rule.Timeout = DefaultTimeout
	}
	c.rules[name] = rule
}

// AddDataToStorer add a data to storer
func (c *Crawler) AddDataToStorer(name string, data interface{}) {
	c.Lock()
	if c.storersLock {
		c.storersTemp[name] = append(c.storersTemp[name], data)
	} else {
		c.storers[name] = append(c.storers[name], data)
	}
	c.Unlock()
}

// Run is an init function of Crawler
func (c *Crawler) Run() {
	c.status = 1
	if !c.option.AutoStopDisable {
		c.Stop()
	}
	go c.loopRequest()
	c.loopStorer()
}

// Stop is a stop function of Crawler
func (c *Crawler) Stop() {
	go func() {
		t := time.Tick(5 * time.Second)
		for range t {
			queuesSize := c.queues.Size() + c.queuesTemp.Size()
			storersSize := 0
			for _, s := range c.storers {
				storersSize += len(s)
			}
			for _, s := range c.storersTemp {
				storersSize += len(s)
			}
			if queuesSize+storersSize == 0 {
				c.status = -1
			}
		}
	}()
}

func (c *Crawler) loopRequest() {
	for {
		if c.status < 0 {
			break
		} else if c.status == 0 {
			continue
		}
		queues := c.queuesTemp
		if c.queuesLock = !c.queuesLock; c.queuesLock {
			queues = c.queues
		}
		queues.Each(func(i int, v interface{}) {
			q := v.(Queue)
			r, ok := c.rules[q.Rule]
			if !ok {
				log.Printf("Crawle[%v] Run() error: rules[%v] is not defined", c.option.Name, q.Rule)
			}
			ctx := &Context{Crawler: c, client: http.Client{Timeout: r.Timeout}, Param: q.Param}
			err := r.do(ctx, q)
			if err != nil {
				log.Printf("Crawle[%v] HTTP error: %v", c.option.Name, err)
			} else if ctx.Document, err = goquery.NewDocumentFromResponse(ctx.Response); err != nil {
				log.Printf("Crawle[%v] HTTP to Document error: %v", c.option.Name, err)
			} else if r.Parse == nil || r.Parse(ctx) {
				queues.Remove(i)
			}
			pauseTime := c.option.PauseTime[0] + randIn(c.option.pauseTimeDif)
			time.Sleep(time.Duration(pauseTime) * time.Millisecond)
		})
		if c.queuesLock {
			c.queues = queues
		} else {
			c.queuesTemp = queues
		}
	}
}

func (c *Crawler) loopStorer() {
	for {
		if c.status < 0 {
			break
		} else if c.status == 0 {
			continue
		}
		c.storersLock = !c.storersLock
		c.RLock()
		storers := c.storersTemp
		if c.storersLock {
			storers = c.storers
		}
		results := c.option.StorerWork(storers)
		c.RUnlock()
		for name, success := range results {
			if success {
				storers[name] = []interface{}{}
			}
		}
		c.Lock()
		if c.storersLock {
			c.storers = storers
		} else {
			c.storersTemp = storers
		}
		c.Unlock()
	}
}
