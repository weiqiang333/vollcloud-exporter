package url_parse

import (
	"fmt"
	"net/url"
)

// GetParameId 获取参数
func GetParameId(urlPath string, idName string) (string, error) {
	if len(idName) == 0 {
		idName = "id"
	}
	u, err := url.Parse(urlPath)
	if err != nil {
		return "", fmt.Errorf("Failed GetParameId %s ", err.Error())
	}
	return u.Query().Get(idName), nil
}
