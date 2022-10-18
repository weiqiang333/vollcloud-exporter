package grab

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"
	"log"
	"net/http"
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

func (s *Services) Get() {
	url := viper.GetString("vollcloud.services.url")
	resp, err := s.HttpClient.Get(url)
	if err != nil {
		log.Fatalln("Failed Services Get error: ", err.Error())
		return
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalln("Failed Services goquery error: ", err.Error())
		return
	}
	s.Doc = doc
}

func (s *Services) GetProductIdUrls() {
	tbody := s.Doc.Find("#tableServicesList tbody")
	urlHref := tbody.Find("td.responsive-edit-button").Each(func(i int, gs *goquery.Selection) {
		title := gs.Find("a")
		href, IsExist := title.Attr("href")
		if IsExist {
			s.IdUrls = append(s.IdUrls, href)
		}
		log.Println("Info GetProductIdUrls() : ", href, IsExist)
	})
	log.Println("Info GetProductIdUrls() IdUrls: ", urlHref.Size(), s.IdUrls)
}
