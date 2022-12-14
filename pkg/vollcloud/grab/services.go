package grab

import (
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"
)

type Services struct {
	HttpClient *http.Client
	Doc        *goquery.Document
	IdUrls     []string
}

func NewServices(httpClient http.Client) *Services {
	return &Services{
		HttpClient: &httpClient,
		IdUrls:     []string{},
	}
}

// Get 获取 services 页面
func (s *Services) Get() {
	url := viper.GetString("vollcloud.services.url")
	resp, err := s.HttpClient.Get(url)
	if err != nil {
		log.Println("Failed Services Get error: ", err.Error())
		return
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Println("Failed Services goquery error: ", err.Error())
		return
	}
	s.Doc = doc
}

// GetProductIdUrls 获取资源的子页面
func (s *Services) GetProductIdUrls() {
	urlHref := s.Doc.Find("#tableServicesList tbody tr").Each(func(i int, gs *goquery.Selection) {
		onclick, IsExist := gs.Attr("onclick")
		if IsExist {
			onclicks := strings.Split(onclick, "'")
			if len(onclick) < 2 {
				log.Println("Warn GetProductIdUrls() unusual Split: ", onclick)
			}
			s.IdUrls = append(s.IdUrls, onclicks[1])
		}
		log.Println("Info GetProductIdUrls() : ", onclick, IsExist)
	})
	log.Println("Info GetProductIdUrls() IdUrls: ", urlHref.Size(), s.IdUrls)
}
