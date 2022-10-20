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

func (p *Productdetails) Get(idUrl string) {
	url := fmt.Sprintf("%s%s&language=english", viper.GetString("vollcloud.productdetails.url"), idUrl)
	resp, err := p.HttpClient.Get(url)
	if err != nil {
		log.Println("Failed Productdetails Get error: ", idUrl, err.Error())
		return
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Println("Failed Productdetails goquery error: ", err.Error())
		return
	}
	p.Doc = doc
}

func (p *Productdetails) GetModuleBody() {
	p.Doc.Find("div.module-body tr").Each(func(i int, s *goquery.Selection) {
		tds := []string{}
		s.Find("td").Each(func(i int, selection *goquery.Selection) {
			tds = append(tds, strings.TrimSpace(selection.Text()))
		})
		//log.Println("Info GetModuleBody", td.Size(), tds)
		p.StatsMapTemp[tds[0]] = tds[1]
	})
	//log.Println("Info GetModuleBody success: ", p.StatsMapTemp)
}

func (p *Productdetails) CreateStats() {
	p.GetModuleBody()
	p.Stats.Hostname = p.StatsMapTemp["Hostname"]
	p.Stats.IpAddress = p.StatsMapTemp["Main IP Address"]
	p.Stats.Status = getStatus(p.StatsMapTemp["Status"])
	p.Stats.Type = p.StatsMapTemp["Type"]
	p.Stats.Memory = p.StatsMapTemp["Memory"]
	p.Stats.Disk = p.StatsMapTemp["HDD"]
	p.getBandwidth(p.StatsMapTemp["Bandwidth"])
	log.Println("Info CreateStats success: ", p.Stats)
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
