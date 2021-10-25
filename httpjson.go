package httpjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

type Caller interface {
	Call(method string, send, recv interface{}) error
}

type Client struct {
	url    string
	client *http.Client
}

type HTTPError struct {
	Code   int
	Status string
	Body   []byte
}

func (a HTTPError) Error() string {
	return fmt.Sprintf("%d: %s", a.Code, a.Status)
}

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func New(client *http.Client, url string, header http.Header) Client {
	defaultTransport := client.Transport
	client.Transport = RoundTripFunc(func(request *http.Request) (*http.Response, error) {
		request.Header = header
		return defaultTransport.RoundTrip(request)
	})
	return Client{
		client: client,
		url:    url,
	}
}

func (c Client) Call(method, path string, send interface{}, recv interface{}) error {
	var (
		b  []byte
		e1 error
	)
	if !isNil(send) {
		b, e1 = json.Marshal(send)
		if e1 != nil {
			return e1
		}
	}
	req, err := http.NewRequest(method, c.url+path, bytes.NewReader(b))
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if _, err1 := io.Copy(&buf, resp.Body); err1 != nil {
		return err1
	}
	if err = resp.Body.Close(); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return HTTPError{
			Code:   resp.StatusCode,
			Status: resp.Status,
			Body:   buf.Bytes(),
		}
	}
	if !isNil(recv) {
		return json.Unmarshal(buf.Bytes(), recv)
	}
	return nil
}

func isNil(v interface{}) bool {
	return v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil())
}

