package grab

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"
)

type Clientarea struct {
	HttpClient *http.Client
	Doc        *goquery.Document
	IdUrls     []string
}

func NewClientarea(httpClient http.Client) *Clientarea {
	return &Clientarea{
		HttpClient: &httpClient,
	}
}

func (c *Clientarea) Get() {
	url := viper.GetString("vollcloud.clientarea.url")
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		log.Println("Failed clientarea Get error: ", err.Error())
		return
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Println("Failed clientarea goquery error: ", err.Error())
		return
	}
	c.Doc = doc
}

func (c *Clientarea) IfUserLogin() (string, error) {
	headerUsername := strings.TrimSpace(c.Doc.Find(".nav-item.dropdown.account .nav-link.dropdown-toggle").Text())

	if len(headerUsername) == 0 {
		pageTitleBox := c.Doc.Find("title").Text() + c.Doc.Find("div.pageError").Text()
		log.Println("Failed user Login, msg: ", strings.Fields(pageTitleBox))
		return "", fmt.Errorf("Failed Login %s ", strings.Fields(pageTitleBox))
	}
	log.Println("Info The current login user is: ", headerUsername)
	return headerUsername, nil
}
