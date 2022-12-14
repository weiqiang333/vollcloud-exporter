package grab

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"

	"vollcloud-exporter/pkg/unit/date"
	"vollcloud-exporter/pkg/unit/url_parse"
)

type Cost struct {
	HttpClient *http.Client
	Doc        *goquery.Document
	CostInfos  []CostInfo
}

func NewCost(httpClient http.Client) *Cost {
	return &Cost{
		HttpClient: &httpClient,
	}
}

type CostInfo struct {
	DateStart      string
	DateEnd        string
	BlendedCostUSD float64
	CostCycle      string
	ProductId      string
}

// GetCost 获取成本页面
func (c *Cost) GetCost() error {
	costUrl := viper.GetString("vollcloud.cost.url")
	resp, err := c.HttpClient.Get(costUrl)
	if err != nil {
		msg := fmt.Sprintf("Failed Cost Get error: %s %s", costUrl, err.Error())
		log.Println(msg)
		return fmt.Errorf(msg)
	}
	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Failed Cost Get StatusCode not is 200, it is %v", resp.StatusCode)
		log.Println(msg)
		return fmt.Errorf(msg)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("Failed Cost goquery error: %s", err.Error())
		log.Println(msg)
		return fmt.Errorf(msg)
	}
	c.Doc = doc
	return nil
}

// GetCostInfos 解析 html 中的成本页面
func (c *Cost) GetCostInfos() {
	c.Doc.Find("div.renewal-bd-table .table tbody tr").Each(func(i int, s *goquery.Selection) {
		costInfo := CostInfo{}
		s.Find("td").Each(func(itd int, std *goquery.Selection) {
			if itd == 4 {
				dateEnd := strings.TrimSpace(std.Text())
				costInfo.DateEnd = dateEnd
			}
			if itd == 5 {
				cycle := strings.TrimSpace(std.Text())
				costInfo.CostCycle = cycle
			}
			if itd == 6 {
				blendedCostUSD := strings.TrimSpace(std.Text())
				usd, err := strconv.ParseFloat(strings.ReplaceAll(strings.ReplaceAll(blendedCostUSD, "$", ""), " USD", ""), 64)
				if err != nil {
					log.Println("Failed GetCostInfos blendedCostUSD Recurring Amount", err.Error())
				}
				costInfo.BlendedCostUSD = usd
			}
			if itd == 7 {
				idUrl, IsExist := std.Find("a").Attr("href")
				if IsExist {
					sid, err := url_parse.GetParameId(idUrl, "sid")
					if err != nil {
						log.Println("Failed GetCostInfos GetParameId", err.Error())
					}
					costInfo.ProductId = sid
				}
			}
			dateStart, err := getDateStart(costInfo.DateEnd, costInfo.CostCycle)
			if err != nil {
				log.Println("Failed GetCostInfos getDateStart", err.Error())
			}
			if len(dateStart) == 0 {
				dateStart = date.GetNowDay()
			}
			costInfo.DateStart = dateStart

			c.CostInfos = append(c.CostInfos, costInfo)
		})
		c.CostInfos = append(c.CostInfos, costInfo)
		c.CostInfos = getCycleCost(c.CostInfos, costInfo)
	})
	log.Println("Info GetCostInfos success: ", len(c.CostInfos))
}

// getCycleCost 获取更多周期账单的成本, 以当前时间为维度, 当期账单, 及以续费后的账单
func getCycleCost(costInfos []CostInfo, costInfo CostInfo) []CostInfo {
	ok, err := date.IfDateBigNow(costInfo.DateStart)
	if err != nil {
		return costInfos
	}
	if ok {
		return costInfos
	}

	dateEnd := costInfo.DateStart
	dateStart, _ := getDateStart(dateEnd, costInfo.CostCycle)
	newCostInfo := CostInfo{
		DateStart:      dateStart,
		DateEnd:        dateEnd,
		BlendedCostUSD: costInfo.BlendedCostUSD,
		CostCycle:      costInfo.CostCycle,
		ProductId:      costInfo.ProductId,
	}
	costInfos = append(costInfos, newCostInfo)
	return getCycleCost(costInfos, newCostInfo)
}

// getDateStart 计算付费周期的起始时间
func getDateStart(dateEnd string, cycle string) (string, error) {
	if cycle == "每年" || cycle == "year" {
		return date.GetDateBeforeYear(dateEnd, -1)
	}
	if cycle == "月" || cycle == "month" {
		return date.GetDateBeforeMonth(dateEnd, -1)
	}
	return date.GetNowDay(), nil
}

// SplitCostCycle 将成本拆分: 年/月/日 (日只返回最近7天的)
func SplitCostCycle(cost CostInfo) []CostInfo {
	var costs []CostInfo
	cycle, err := date.GetDateSubPeriodUnit(cost.DateStart, cost.DateEnd)
	if err != nil {
		log.Println("Failed SplitCostCycle error", cycle, err.Error())
		return costs
	}
	costs = append(costs, CostInfo{
		DateStart:      cost.DateStart,
		DateEnd:        cost.DateEnd,
		BlendedCostUSD: cost.BlendedCostUSD,
		CostCycle:      cycle,
	})
	if cycle == "year" || cycle == "每年" {
		newCycle := "month"
		costDay := cost.BlendedCostUSD / 365
		dateRangeMonth, err := date.GetDateRangeYearToMonth(cost.DateStart, cost.DateEnd)
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
			costs = append(costs, CostInfo{
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
			costs = append(costs, CostInfo{
				DateStart:      d,
				DateEnd:        dateRangeDays[i+1],
				BlendedCostUSD: costDay,
				CostCycle:      newCycle,
			})
		}
	}
	if cycle == "month" || cycle == "每月" {
		newCycle := "day"
		costDay := cost.BlendedCostUSD / 30
		dateRangeDays := date.GetDateRangeToDay(date.GetBeforeDay(-7), date.GetNowDay())
		for i, d := range dateRangeDays {
			if i+1 >= len(dateRangeDays) {
				continue
			}
			costs = append(costs, CostInfo{
				DateStart:      d,
				DateEnd:        dateRangeDays[i+1],
				BlendedCostUSD: costDay,
				CostCycle:      newCycle,
			})
		}
	}
	return costs
}
