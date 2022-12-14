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
)

type Productdetails struct {
	HttpClient   *http.Client
	Doc          *goquery.Document
	Stats        Stats
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

func NewProductdetails(httpClient http.Client) *Productdetails {
	return &Productdetails{
		HttpClient:   &httpClient,
		Stats:        Stats{},
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

// GetModuleBody 将资源 tr 中的信息临时存放至 StatsMapTemp，方便提取
func (p *Productdetails) GetModuleBody() {
	p.Doc.Find("div.module-body .table.pm-stats tr").Each(func(i int, s *goquery.Selection) {
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
	hostname := strings.TrimSpace(p.Doc.Find("#solus-hostname").Text())
	if len(hostname) == 0 {
		hostname = "nil"
	}
	status := strings.TrimSpace(p.Doc.Find("#solus-hostname").Text())
	if len(status) == 0 {
		status = "nil"
	}
	p.StatsMapTemp["Hostname"] = hostname
	p.StatsMapTemp["Status"] = status
	p.Doc.Find("div.svm-header-config div").Each(func(i int, s *goquery.Selection) {
		conf := strings.Split(strings.TrimSpace(s.Text()), ":")
		if len(conf) >= 2 {
			p.StatsMapTemp[strings.TrimSpace(conf[0])] = strings.TrimSpace(conf[1])
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
	if s == "online" || s == "Online" {
		return 1
	}
	return 0
}
