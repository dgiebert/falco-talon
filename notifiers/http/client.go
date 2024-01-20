package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"

	"github.com/Issif/falco-talon/configuration"
	"github.com/Issif/falco-talon/utils"
)

var ErrHeaderMissing = errors.New("header missing")                   // ErrHeaderMissing = 400
var ErrClientAuthenticationError = errors.New("authentication error") // ErrClientAuthenticationError = 401
var ErrForbidden = errors.New("access denied")                        // ErrForbidden = 403
var ErrNotFound = errors.New("resource not found")                    // ErrNotFound = 404
var ErrUnprocessableEntityError = errors.New("wrong request")         // ErrUnprocessableEntityError = 422
var ErrTooManyRequest = errors.New("exceeding post rate limit")       // ErrTooManyRequest = 429

const DefaultContentType = "application/json; charset=utf-8"
const DefaultHTTPMethod = "POST"
const DefaultUserAgent = "Falco-Talon"

type Client struct {
	Headers    http.Header
	HTTPMethod string
	Compressed bool
}

func CheckURL(u string) error {
	reg := regexp.MustCompile(`(http)(s?)://.*`)
	if !reg.MatchString(u) {
		return errors.New("invalid url")
	}

	if _, err := url.ParseRequestURI(u); err != nil {
		return errors.New("invalid url")
	}

	_, err := url.Parse(u)
	if err != nil {
		return err
	}

	return nil
}

func DefaultClient() *Client {
	h := http.Header{}
	h.Set("Content-Type", DefaultContentType)
	return &Client{
		HTTPMethod: "POST",
		Headers:    h,
	}
}

func NewClient(httpMethod, contentType, userAgent string, headers map[string]string) *Client {
	h := http.Header{}
	if len(headers) != 0 {
		for i, j := range headers {
			h.Add(i, j)
		}
	}

	m := DefaultHTTPMethod
	if httpMethod != "" {
		m = httpMethod
	}

	a := DefaultUserAgent
	if userAgent != "" {
		a = userAgent
	}
	h.Set("User-Agent", a)

	ct := DefaultContentType
	if contentType != "" {
		ct = contentType
	}
	h.Set("Content-Type", ct)

	return &Client{
		HTTPMethod: m,
		Headers:    h,
	}
}

func (c *Client) Post(u string, payload interface{}) error {
	// defer + recover to catch panic if output doesn't respond
	config := configuration.GetConfiguration()
	defer func() {
		if err := recover(); err != nil {
			utils.PrintLog("error", config.LogFormat, utils.LogLine{Error: "recover"})
		}
	}()

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(payload); err != nil {
		return err
	}

	client := &http.Client{
		Transport: http.DefaultTransport.(*http.Transport).Clone(),
	}

	req, err := http.NewRequest(c.HTTPMethod, u, body)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent: // 200, 201, 202, 204
		return nil
	case http.StatusBadRequest: // 400
		return ErrHeaderMissing
	case http.StatusUnauthorized: // 401
		return ErrClientAuthenticationError
	case http.StatusForbidden: // 403
		return ErrForbidden
	case http.StatusNotFound: // 404
		return ErrNotFound
	case http.StatusUnprocessableEntity: // 422
		return ErrUnprocessableEntityError
	case http.StatusTooManyRequests: // 429
		return ErrTooManyRequest
	default:
		return errors.New(resp.Status)
	}
}
