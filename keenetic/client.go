package keenetic

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	User       string
	Password   string
	HTTPClient *http.Client
}

func NewClient(ip, user, password string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(ip, "http") {
		ip = "http://" + ip
	}
	ip = strings.TrimRight(ip, "/")

	return &Client{
		BaseURL:  ip,
		User:     user,
		Password: password,
		HTTPClient: &http.Client{
			Jar:     jar,
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *Client) Login() error {
	authURL := c.BaseURL + "/auth"
	log.Printf("[Keenetic] Auth: %s", authURL)

	req, _ := http.NewRequest("GET", authURL, nil)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}
	if resp.StatusCode != 401 {
		return fmt.Errorf("auth init failed: %d", resp.StatusCode)
	}

	realm := resp.Header.Get("X-NDM-Realm")
	challenge := resp.Header.Get("X-NDM-Challenge")

	md5Str := fmt.Sprintf("%s:%s:%s", c.User, realm, c.Password)
	md5Hash := md5.Sum([]byte(md5Str))
	md5Hex := hex.EncodeToString(md5Hash[:])

	shaStr := challenge + md5Hex
	shaHash := sha256.Sum256([]byte(shaStr))
	shaHex := hex.EncodeToString(shaHash[:])

	authData := map[string]string{
		"login":    c.User,
		"password": shaHex,
	}
	jsonData, _ := json.Marshal(authData)

	respAuth, err := c.HTTPClient.Post(authURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer respAuth.Body.Close()

	if respAuth.StatusCode != 200 {
		return fmt.Errorf("auth failed: %d", respAuth.StatusCode)
	}
	return nil
}

func (c *Client) RciGetRaw(path string) ([]byte, error) {
	data, err := c.doRequestRaw("GET", path, nil)
	if err != nil && strings.Contains(err.Error(), "401") {
		_ = c.Login()
		return c.doRequestRaw("GET", path, nil)
	}
	return data, err
}

func (c *Client) RciGet(path string, target interface{}) error {
	data, err := c.RciGetRaw(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (c *Client) RciPost(path string, payload interface{}) error {
	if payload == nil {
		payload = map[string]interface{}{}
	}

	_, err := c.doRequestRaw("POST", path, payload)
	if err != nil && strings.Contains(err.Error(), "401") {
		_ = c.Login()
		_, err = c.doRequestRaw("POST", path, payload)
	}
	return err
}

func (c *Client) SendBatch(payload interface{}, response interface{}) error {
	url := c.BaseURL + "/rci/"

	err := c.doPost(url, payload, response)
	if err != nil && strings.Contains(err.Error(), "401") {
		_ = c.Login()
		return c.doPost(url, payload, response)
	}
	return err
}

func (c *Client) doPost(url string, payload interface{}, response interface{}) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	log.Printf("[RCI BATCH POST] URL=%s payload=%s", url, string(jsonBytes))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		log.Printf("[RCI BATCH POST] URL=%s status=401", url)
		return fmt.Errorf("401 Unauthorized")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[RCI BATCH POST] URL=%s status=%d body=%s", url, resp.StatusCode, string(body))
		return fmt.Errorf("API Error %d: %s", resp.StatusCode, string(body))
	}

	if response != nil {
		return json.NewDecoder(resp.Body).Decode(response)
	}
	return nil
}

func (c *Client) doRequestRaw(method, path string, payload interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s/rci/%s", c.BaseURL, path)

	var body io.Reader
	var payloadDump string
	if payload != nil {
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonBytes)
		payloadDump = string(jsonBytes)
	}

	if method == "POST" {
		log.Printf("[RCI POST] URL=%s payload=%s", url, payloadDump)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if method == "POST" {
		log.Printf("[RCI POST] URL=%s status=%d body=%s", url, resp.StatusCode, string(data))
	}

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("401 Unauthorized")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API Error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}