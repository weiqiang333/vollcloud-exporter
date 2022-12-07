package grab

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"

	"vollcloud-exporter/pkg/unit/conversion"
	"vollcloud-exporter/pkg/unit/date"
)

type Productdetails struct {
	HttpClient   *http.Client
	Doc          *goquery.Document
	Stats        Stats
	Cost         Cost
	StatsMapTemp map[string]string
}

type Stats struct {
	Hostname         string
	IpAddress        string
	Status           float64
	Type             string
	Memory           string
	Disk             string
	BandwidthTotalGB float64
	BandwidthUsedGB  float64
	BandwidthFreeGB  float64
	BandwidthUsage   float64 // 使用百分比
}

type Cost struct {
	DateStart      string
	DateEnd        string
	BlendedCostUSD float64
	CostCycle      string
}

func NewProductdetails(httpClient http.Client) *Productdetails {
	return &Productdetails{
		HttpClient:   &httpClient,
		Stats:        Stats{},
		Cost:         Cost{},
		StatsMapTemp: map[string]string{},
	}
}

// Get 访问资源页面，获取资源描述.
func (p *Productdetails) Get(idUrl string) error {
	url := fmt.Sprintf("%s%s&language=english", viper.GetString("vollcloud.productdetails.url"), idUrl)
	resp, err := p.HttpClient.Get(url)
	if err != nil {
		msg := fmt.Sprintf("Failed Productdetails Get error: %s %s", idUrl, err.Error())
		log.Println(msg)
		return fmt.Errorf(msg)
	}
	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Failed Productdetails Get StatusCode not is 200, it is %v", resp.StatusCode)
		log.Println(msg)
		return fmt.Errorf(msg)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("Failed Productdetails goquery error: %s", err.Error())
		log.Println(msg)
		return fmt.Errorf(msg)
	}
	p.Doc = doc
	return nil
}

// GetProductDetails 解析 html 中的产品详情信息
func (p *Productdetails) GetProductDetails() error {
	productDetails := p.Doc.Find("div.product-details .col-md-6.text-center")
	if productDetails.Length() == 0 {
		return fmt.Errorf("Failed GetProductDetails is nil")
	}
	m := formatProductDetailsText(productDetails.Text())
	log.Println("Info GetProductDetails formatProductDetailsText: ", m)
	if len(m) == 0 {
		return fmt.Errorf("Failed GetProductDetails is nil")
	}
	dateStart, err := date.ChangeDateLayout(m["Registration Date"])
	if err != nil {
		return fmt.Errorf("Failed GetProductDetails %s ", err.Error())
	}
	dateEnd, err := date.ChangeDateLayout(m["Next Due Date"])
	if err != nil {
		return fmt.Errorf("Failed GetProductDetails %s ", err.Error())
	}
	blendedCostUSD, err := strconv.ParseFloat(strings.ReplaceAll(strings.ReplaceAll(m["Recurring Amount"], "$", ""), " USD", ""), 64)
	if err != nil {
		return fmt.Errorf("Failed GetProductDetails Recurring Amount %s ", err.Error())
	}
	p.Cost = Cost{
		DateStart:      dateStart,
		DateEnd:        dateEnd,
		BlendedCostUSD: blendedCostUSD,
	}
	return nil
}

// GetModuleBody 将资源 tr 中的信息临时存放至 StatsMapTemp，方便提取
func (p *Productdetails) GetModuleBody() {
	p.Doc.Find("div.module-body tr").Each(func(i int, s *goquery.Selection) {
		tds := []string{}
		s.Find("td").Each(func(i int, selection *goquery.Selection) {
			if strings.TrimSpace(selection.Text()) == "" {
				tds = append(tds, "nil")
			} else {
				tds = append(tds, strings.TrimSpace(selection.Text()))
			}
		})
		if len(tds) >= 2 {
			//log.Println("Info GetModuleBody", tds)
			p.StatsMapTemp[tds[0]] = tds[1]
		}
	})
	//log.Println("Info GetModuleBody success: ", p.StatsMapTemp)
}

// CreateStats 将资源页面的信息进行统计拼凑
func (p *Productdetails) CreateStats() error {
	p.GetModuleBody()
	if len(p.StatsMapTemp) <= 2 {
		msg := fmt.Sprintf("Failed CreateStats in GetModuleBody's StatsMapTemp is not to standard. StatsMapTemp: %s", p.StatsMapTemp)
		log.Println(msg)
		return fmt.Errorf(msg)
	}
	p.Stats.Hostname = p.StatsMapTemp["Hostname"]
	p.Stats.IpAddress = p.StatsMapTemp["Main IP Address"]
	p.Stats.Status = getStatus(p.StatsMapTemp["Status"])
	p.Stats.Type = p.StatsMapTemp["Type"]
	p.Stats.Memory = p.StatsMapTemp["Memory"]
	p.Stats.Disk = p.StatsMapTemp["HDD"]
	if b, ok := p.StatsMapTemp["Bandwidth"]; ok {
		p.getBandwidth(b)
	} else {
		msg := fmt.Sprintf("Failed CreateStats in get StatsMapTemp[\"Bandwidth\"], key not exists")
		log.Println(msg)
		return fmt.Errorf(msg)
	}
	log.Println("Info CreateStats success: ", p.Stats)
	return nil
}

// SplitCostCycle 将成本拆分: 年/月/日 (日只返回最近7天的)
func (p *Productdetails) SplitCostCycle() []Cost {
	var costs []Cost
	cycle, err := date.GetDateSubPeriodUnit(p.Cost.DateStart, p.Cost.DateEnd)
	if err != nil {
		log.Println("Failed SplitCostCycle error", cycle, err.Error())
		return costs
	}
	costs = append(costs, Cost{
		DateStart:      p.Cost.DateStart,
		DateEnd:        p.Cost.DateEnd,
		BlendedCostUSD: p.Cost.BlendedCostUSD,
		CostCycle:      cycle,
	})
	if cycle == "year" {
		newCycle := "month"
		costDay := p.Cost.BlendedCostUSD / 365
		dateRangeMonth, err := date.GetDateRangeYearToMonth(p.Cost.DateStart, p.Cost.DateEnd)
		if err != nil {
			log.Println("Warn GetDateRangeYearToMonth", dateRangeMonth, err.Error())
			return costs
		}
		for i, m := range dateRangeMonth {
			if i+1 >= len(dateRangeMonth) {
				continue
			}
			day, err := date.GetDateSubPeriodDays(m, dateRangeMonth[i+1])
			if err != nil {
				log.Println("Warn GetDateSubPeriodDays", err.Error())
				continue
			}
			costs = append(costs, Cost{
				DateStart:      m,
				DateEnd:        dateRangeMonth[i+1],
				BlendedCostUSD: day * costDay,
				CostCycle:      newCycle,
			})
		}
		newCycle = "day"
		dateRangeDays := date.GetDateRangeToDay(date.GetBeforeDay(-7), date.GetNowDay())
		for i, d := range dateRangeDays {
			if i+1 >= len(dateRangeDays) {
				continue
			}
			costs = append(costs, Cost{
				DateStart:      d,
				DateEnd:        dateRangeDays[i+1],
				BlendedCostUSD: costDay,
				CostCycle:      newCycle,
			})
		}
	}
	if cycle == "month" {
		newCycle := "day"
		costDay := p.Cost.BlendedCostUSD / 30
		dateRangeDays := date.GetDateRangeToDay(date.GetBeforeDay(-7), date.GetNowDay())
		for i, d := range dateRangeDays {
			if i+1 >= len(dateRangeDays) {
				continue
			}
			costs = append(costs, Cost{
				DateStart:      d,
				DateEnd:        dateRangeDays[i+1],
				BlendedCostUSD: costDay,
				CostCycle:      newCycle,
			})
		}
	}
	return costs
}

// getBandwidth - b 例子: "254.38 GB of 1000 GB Used / 745.62 GB Free\n\n\n                                25%"
func (p *Productdetails) getBandwidth(bandwidth string) {
	sOf := strings.Split(bandwidth, " of ")
	sUsed := strings.Split(sOf[1], " Used / ")
	sFree := strings.Split(sUsed[1], " Free")
	used := sOf[0]
	total := sUsed[0]
	free := sFree[0]
	usage, _ := strconv.ParseFloat(strings.Split(strings.TrimSpace(sFree[1]), "%")[0], 64)
	//log.Println(fmt.Sprintf("Info getBandwidth; used: %s, total: %s, free: %s, usage: %v", used, total, free, usage))

	if usedGB, err := getConversion(used); err == nil {
		p.Stats.BandwidthUsedGB = usedGB
	}
	if totalGB, err := getConversion(total); err == nil {
		p.Stats.BandwidthTotalGB = totalGB
	}
	if freeGB, err := getConversion(free); err == nil {
		p.Stats.BandwidthFreeGB = freeGB
	}
	p.Stats.BandwidthUsage = usage
}

func getConversion(s string) (float64, error) {
	ss := strings.Fields(s)
	n, err := strconv.ParseFloat(ss[0], 64)
	if err != nil {
		log.Println("Failed getConversion string to ParseFloat error:", s, err.Error())
		return n, fmt.Errorf("Failed getConversion string to ParseFloat error: %v, %s", s, err.Error())
	}
	if ss[1] == "GB" {
		return n, nil
	}
	if ss[1] == "MB" {
		return conversion.MBtoGB(n), nil
	}
	if ss[1] == "TB" {
		return conversion.TBtoGB(n), nil
	}
	return n, fmt.Errorf("Failed getConversion to get size type no existent, %v", s)
}

func getStatus(s string) float64 {
	if s == "online" {
		return 1
	}
	return 0
}

// formatProductDetailsText 格式化解析项目信息的数据 -> map
func formatProductDetailsText(text string) map[string]string {
	texts := strings.Split(text, "\n")
	data := map[string]string{}
	sub := []string{}
	for _, v := range texts {
		v = strings.TrimSpace(v)
		if len(v) == 0 {
			sub = []string{}
			continue
		}
		sub = append(sub, v)
		if len(sub) < 2 {
			continue
		}
		data[sub[0]] = sub[1]
		sub = []string{}
	}
	return data
}
