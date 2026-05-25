package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const signatureMethod = "HmacSHA256"

type Client struct {
	baseURL   string
	appID     string
	appSecret string
	http      *http.Client
	onCall    func()
}

type Response struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func New(baseURL, appID, appSecret string, onCall func()) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		appID:     appID,
		appSecret: appSecret,
		http:      &http.Client{Timeout: 30 * time.Second},
		onCall:    onCall,
	}
}

func (c *Client) Get(path string, params url.Values) (Response, error) {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return Response{}, err
	}
	for k, v := range c.sign(http.MethodGet, path) {
		req.Header.Set(k, v)
	}
	if c.onCall != nil {
		c.onCall()
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	var out Response
	if err := json.Unmarshal(body, &out); err != nil {
		return Response{}, err
	}
	if out.Code != 0 {
		return out, fmt.Errorf("api code %d", out.Code)
	}
	return out, nil
}

func (c *Client) sign(method, path string) map[string]string {
	clean := strings.TrimPrefix(strings.Split(path, "?")[0], "/")
	parts := strings.Split(strings.TrimRight(clean, "/"), "/")
	reqPath := parts[len(parts)-1]
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	nonce := randomNonce()
	stringToSign := ts + "/" + nonce + "/" + c.appID + "/" + reqPath + "/" + method + "/" + signatureMethod
	mac := hmac.New(sha256.New, []byte(c.appSecret))
	_, _ = mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return map[string]string{
		"X-CA-AppId":            c.appID,
		"X-CA-Timestamp":        ts,
		"X-CA-Nonce":            nonce,
		"X-CA-Signature-Method": signatureMethod,
		"X-CA-Signature":        signature,
	}
}

func randomNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(b)
}
