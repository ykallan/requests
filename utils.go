package requests

import (
	"github.com/mitchellh/mapstructure"
	"net/http"
	"time"
)

func TransferCookies(_cookies []map[string]interface{}) []*http.Cookie {
	cookies := make([]*http.Cookie, 0)
	for _, cookie := range _cookies {
		var _cookie http.Cookie
		_ = mapstructure.Decode(cookie, &_cookie)
		if _, ok := cookie["Expire"]; ok {
			_cookie.Expires = time.Unix(cookie["Expire"].(int64), 0)
		} else {
			_cookie.Expires = time.Now()
		}
		cookies = append(cookies, &_cookie)
	}
	return cookies
}
