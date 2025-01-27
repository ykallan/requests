package requests

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"github.com/axgle/mahonia"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

type Response struct {
	*http.Response
	encoding string
	cost     time.Duration
	Text     string
	Bytes    []byte
	err      error
	xpathResult []string
}

var validStatusCode = [...]int{
	http.StatusOK, http.StatusCreated,
}

func NewResponse(r *http.Response, cost time.Duration) *Response {
	resp := &Response{
		Response: r,
		encoding: "utf-8",
		cost:     cost,
		Text:     "",
		Bytes:    []byte{},
	}

	err := resp.bytes()
	resp.err = err
	resp.text()
	return resp
}

func (r *Response) text() {
	r.Text = string(r.Bytes)
}

func (r *Response) bytes() error {
	defer r.Body.Close()

	var err error
	r.Bytes, err = ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewBuffer(r.Bytes))
	return err
}

func (r *Response) Error() error {
	return r.err
}

func (r *Response) Cost() time.Duration {
	return r.cost
}

func (r *Response) XML(v interface{}) error {
	return xml.Unmarshal(r.Bytes, v)
}

func (r *Response) Document() (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(strings.NewReader(r.Text))
}

func (r *Response) ParseInputForm(doc *goquery.Document, id string) Form {
	data := Form{}
	doc.Find("#" + id).Find("input").Each(func(i int, c *goquery.Selection) {
		key, exit := c.Attr("name")
		value, exit1 := c.Attr("value")
		if exit && exit1 && key != "" {
			data[key] = value
		}
	})
	return data
}

func (r *Response) JSON() (gjson.Result, error) {
	if r.err != nil {
		return gjson.Result{}, r.err
	}
	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		return gjson.Result{}, errors.New("invalid response code: " + strconv.Itoa(r.StatusCode))
	}
	return gjson.ParseBytes(r.Bytes), nil
}

func (r *Response) CallbackJSON() (gjson.Result, error) {
	if r.err != nil {
		return gjson.Result{}, r.err
	}
	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		return gjson.Result{}, errors.New("invalid response code: " + strconv.Itoa(r.StatusCode))
	}
	re, _ := regexp.Compile(`\(\s*{[\s\S]*?}\s*\)`)
	y := re.FindStringSubmatch(r.Text)
	if len(y) == 0 {
		return gjson.Result{}, ErrNotJSONResponse
	}
	return gjson.ParseBytes([]byte(y[0][1 : len(y[0])-1])), nil
}

func (r *Response) SetEncode(e string) error {
	if r.encoding != e {
		r.encoding = strings.ToLower(e)
		decoder := mahonia.NewDecoder(e)
		if decoder == nil {
			return ErrUnrecognizedEncoding
		}
		r.Text = decoder.ConvertString(r.Text)
	}
	return nil
}

func (r *Response) GetEncode() string {
	return r.encoding
}

func (r *Response) SaveFile(filename string) error {
	dst, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = dst.Write(r.Bytes)
	if err != nil {
		return err
	}
	return nil
}

func (r *Response) Location() string {
	return r.Header.Get("Location")
}

func (r *Response) Cookie(key string) string {
	for _, cookie := range r.Cookies() {
		if cookie.Name == key {
			return cookie.Value
		}
	}
	return ""
}

func (r *Response) Xpath(expr string) *Response {
	r.xpathResult = make([]string, 0)
	doc, err := htmlquery.Parse(strings.NewReader(r.Text))
	if err != nil {
		fmt.Println(err.Error())
		return r
	}
	for _, node := range htmlquery.Find(doc, expr) {
		if len(node.Data) > 0{
			r.xpathResult = append(r.xpathResult, node.Data)
		}
	}
	return r
}

func (r *Response) Get() (string, bool) {
	if len(r.xpathResult) ==0 {
		return "", false
	}
	return r.xpathResult[0], true
}

func (r *Response) Getall() []string {
	return r.xpathResult
}

