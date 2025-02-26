package spider

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"time"
	"sync"
	"os"
	"strings"
	"encoding/json"
)

type PageInfo struct {
	Title       string
	Description string
	Links       []string
}

func CrawlingCommodityData() {
	c := colly.NewCollector(
		colly.AllowedDomains("www.zol.com.cn", "zol.com.cn"), // 限制域名
		colly.Async(true),                   // 启用异步
	)

	// 配置爬虫参数
	c.Limit(&colly.LimitRule{
		Parallelism: 2,               // 并发数
		RandomDelay: 1 * time.Second, // 请求间隔
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Printf("Visiting %s\n", r.URL)
	})

	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Request error:", err)
	})

	// 创建存储结构
	var pageData PageInfo

	// 解析 meta 信息
	c.OnHTML("head", func(h *colly.HTMLElement) {
		pageData.Title = h.ChildText("title")
		pageData.Description = h.ChildAttr("meta[name='description']", "content")
	})

	// 收集所有链接
	c.OnHTML("a[href]", func(h *colly.HTMLElement) {
		fmt.Println("=== ", h)
		link := h.Attr("href")
		if link != "" {
			pageData.Links = append(pageData.Links, link)
		}
	})

	// 完成回调
	c.OnScraped(func(r *colly.Response) {
		fmt.Println("\n=== 抓取结果 ===")
		fmt.Printf("标题: %s\n描述: %s\n发现链接数: %d\n",
			pageData.Title,
			pageData.Description,
			len(pageData.Links))
	})

	// 开始抓取
	c.Visit("https://www.zol.com.cn/")
	fmt.Println("=== 开始抓取 ===")
	c.Wait() // 等待异步任务完成
}


type Commodity struct {
	URL              string 	`json:"url"`
	Title            string 	`json:"title"`
	Price            string 	`json:"price"`
	Specs            []string 	`json:"specs"`
	Description      string 	`json:"description"`
	ImagesURLs       []string 	`json:"images_urls"`
	CrawledAt        time.Time `json:"crawled_at"`
}

type SafeWriter struct {
	mu sync.Mutex
	f *os.File
}

func createOutputFile(filename string) *os.File {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	return file
}

func isProductLink(url string) bool {
	return strings.Contains(url, "/product/") || strings.Contains(url, "/detail_/")
}

func extractPrice(s string) string {
	return strings.TrimSpace(strings.NewReplacer("￥", "", " ", "", "\n", "").Replace(s))
}


func (sw *SafeWriter) WriteData(c Commodity) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	data, _ := json.MarshalIndent(c, "  ", "  ")
	if _, err := sw.f.WriteString("  " + string(data) + ",\n"); err != nil {
		fmt.Println("写入文件失败：", err)
	}
}

func CrawlingCommodityData2() {
	writer := &SafeWriter{
		f: createOutputFile("commodity.json"),
	}
	defer writer.f.Close()


	c := colly.NewCollector(
		colly.AllowedDomains("www.zol.com.cn", "zol.com.cn"), // 限制域名
		colly.Async(true),                   // 启用异步
		colly.MaxDepth(3),
		colly.CacheDir("./cache"),
	)
	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	// 配置爬虫参数
	c.Limit(&colly.LimitRule{
		Parallelism: 4,               // 并发数
		DomainGlob: "*.zol.com.cn*",
		Delay: 1 * time.Second,
		RandomDelay: 500 * time.Millisecond,
	})

	c.OnError(func(r *colly.Response, err error){
		fmt.Println("=== 抓取失败 ===", r.StatusCode)
		fmt.Printf("请求失败：URL：%s 错误：%v\n", r.Request.URL, err)
		if r.StatusCode == 429 {
			fmt.Println("处罚反爬机制，等待 10 秒。。。")
			time.Sleep(10 * time.Second)
		}
	})
	detailCollector := c.Clone()
	detailCollector.OnHTML("div.main-container", func(h *colly.HTMLElement){
		commodity := Commodity{
			URL: h.Request.URL.String(),
			Title: strings.TrimSpace(h.ChildText("h1.product-name")),
			Price: extractPrice(h.ChildText("div.price-box")),
			Description: h.ChildAttr("meta[name='description']", "content"),
			CrawledAt: time.Now(),
		}
		h.ForEach("div.spec-item", func(_ int, el *colly.HTMLElement){
			commodity.Specs = append(commodity.Specs, fmt.Sprintf("%s: %s", 
			el.ChildText("span.spec-title"), 
			el.ChildText("span.spec-value")))
		})
		h.ForEach("div.gallery-img img", func(_ int, el *colly.HTMLElement){
			if src := el.Attr("data-src"); src != "" {
				commodity.ImagesURLs = append(commodity.ImagesURLs, h.Request.AbsoluteURL(src))
			}
		})
		writer.WriteData(commodity)
	})
	c.OnHTML("a[href]", func(h *colly.HTMLElement){
		link := h.Request.AbsoluteURL(h.Attr("href"))
		if (isProductLink(link)) {
			_ = detailCollector.Visit(link)
		}
	})
	c.Visit("https://www.zol.com.cn/")
	c.Wait()
	fmt.Println("=== 抓取完成 ===")
}