package login

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

type Login struct {
	Url          string
	UrlValues    url.Values
	CookiejarJar cookiejar.Jar
	Timeout      time.Duration
	HttpClient   http.Client
}

func NewLogin() *Login {
	return &Login{}
}

func (l *Login) Login() {

}
