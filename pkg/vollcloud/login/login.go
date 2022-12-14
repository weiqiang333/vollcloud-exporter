package login

import (
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/viper"
	"golang.org/x/net/publicsuffix"
)

type Login struct {
	Url        string
	UrlValues  url.Values
	Timeout    time.Duration
	HttpClient *http.Client
}

func NewLogin() *Login {
	loginUrl := viper.GetString("vollcloud.login.url")
	username := viper.GetString("vollcloud.login.username")
	password := viper.GetString("vollcloud.login.password")
	timeout := viper.GetString("vollcloud.timeout")
	timeout64, _ := strconv.ParseInt(timeout, 10, 64)
	urlValues := url.Values{
		"username": []string{username},
		"password": []string{password},
	}
	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, _ := cookiejar.New(&options)
	client := &http.Client{
		Jar:     jar,
		Timeout: time.Duration(timeout64) * time.Second,
	}
	return &Login{
		Url:        loginUrl,
		UrlValues:  urlValues,
		Timeout:    time.Duration(timeout64) * time.Second,
		HttpClient: client,
	}
}

// Login res username, error
func (l *Login) Login() (string, error) {
	client := l.HttpClient
	resp, err := client.PostForm(l.Url, l.UrlValues)
	if err != nil {
		log.Println("Failed Login in err: ", err.Error())
		return "", err
	}
	//body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Println("Failed goquery error: ", err)
		return "", err
	}
	//fmt.Println(doc.Text(), doc.Find(".nav-item.dropdown.account").Text(), doc.Find(".nav-item.dropdown.account .nav-link.dropdown-toggle").Text())
	headerUsername := strings.TrimSpace(doc.Find(".nav-item.dropdown.account .nav-link.dropdown-toggle").Text())

	log.Println("Info login success user: ", headerUsername)
	if len(headerUsername) == 0 {
		pageTitleBox := doc.Find("title").Text() + doc.Find("div.pageError").Text() + doc.Find("#MGAlerts").Text()
		log.Println("Failed Login, msg: ", strings.Fields(pageTitleBox))
		return "", fmt.Errorf("Failed Login %s ", strings.Fields(pageTitleBox))
	}
	return headerUsername, nil
}
