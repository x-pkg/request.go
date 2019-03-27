package requests

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type Request struct {
	TLSConfig     *tls.Config
	Retry         int
	RetryDuration time.Duration
	Header        http.Header
	Response      Response
}

type Response struct {
	Body       []byte
	Header     http.Header
	StatusCode int
}

func New() *Request {
	return &Request{
		TLSConfig:     nil,
		Header:        make(http.Header),
		Retry:         10,
		RetryDuration: 10,
	}
}

func (r *Request) request(url string, method string, body interface{}) error {
	var (
		err  error
		resp *http.Response
	)
	tr := &http.Transport{
		TLSClientConfig: r.TLSConfig,
	}
	client := &http.Client{Transport: tr}

	var rc io.Reader
	switch k := reflect.TypeOf(body).Kind(); k {
	case
		reflect.String,
		reflect.Map:
		jsonValue, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rc = bytes.NewBuffer(jsonValue)
	default:
		switch body.(type) {
		case *strings.Reader:
			rc = body.(*strings.Reader)
		case io.Reader:
			rc = body.(io.Reader)
		default:
			errors.New("unexpected body input")
		}
	}

	req, err := http.NewRequest(method, url, rc)
	if err != nil {
		return err
	}
	req.Header = r.Header
	err = Retry(r.Retry, r.RetryDuration*time.Second, func() error {
		resp, err = client.Do(req)
		return err
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	r.Response.Body, err = ioutil.ReadAll(resp.Body)
	r.Response.Header = resp.Header
	r.Response.StatusCode = resp.StatusCode
	return nil
}

func Retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if err != nil {
			if attempts == 0 {
				time.Sleep(sleep)
				return Retry(attempts, sleep, f)
			}
			if attempts--; attempts > 0 {
				time.Sleep(sleep)
				return Retry(attempts, sleep, f)
			}
			return err
		}
	}
	return nil
}

func (r *Request) Post(url string, body interface{}) error {
	return r.request(url, "POST", body)
}

func (r *Request) Get(url string) error {
	return r.request(url, "Get", "")
}

func (r *Request) Put(url string, body interface{}) error {
	return r.request(url, "PUT", body)
}

func (r *Request) Delete(url string, body interface{}) error {
	return r.request(url, "DELETE", body)
}

func (r *Request) Json() (map[string]interface{}, error) {
	var data map[string]interface{}
	err := json.Unmarshal(r.Response.Body, &data)
	return data, err
}
