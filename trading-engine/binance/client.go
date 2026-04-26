package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type Client struct {
	baseURL      string
	apiKey       string
	apiSecret    string
	httpClient   *http.Client
	timeOffset   int64
	timeOnce     sync.Once
	mu           sync.Mutex
	symbolInfo   map[string]SymbolInfo
	symbolInfoMu sync.RWMutex
}

type SymbolInfo struct {
	StepSize     decimal.Decimal
	MinQty       decimal.Decimal
	TickSize     decimal.Decimal
	PricePrecision int
	QtyPrecision   int
}

type OrderResponse struct {
	OrderID    int64  `json:"orderId"`
	Symbol     string `json:"symbol"`
	Status     string `json:"status"`
	Side       string `json:"side"`
	Type       string `json:"type"`
	AvgPrice   string `json:"avgPrice"`
	ExecutedQty string `json:"executedQty"`
	OrigQty    string `json:"origQty"`
}

func NewClient(baseURL, apiKey, apiSecret string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		symbolInfo: make(map[string]SymbolInfo),
	}
}

func (c *Client) syncTime() {
	resp, err := c.httpClient.Get(c.baseURL + "/fapi/v1/time")
	if err != nil {
		log.Printf("time sync failed: %v", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		ServerTime int64 `json:"serverTime"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("time sync decode failed: %v", err)
		return
	}
	c.timeOffset = result.ServerTime - time.Now().UnixMilli()
}

func (c *Client) timestamp() int64 {
	c.timeOnce.Do(c.syncTime)
	return time.Now().UnixMilli() + c.timeOffset
}

func (c *Client) sign(params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+params.Get(k))
	}
	query := strings.Join(parts, "&")
	mac := hmac.New(sha256.New, []byte(c.apiSecret))
	mac.Write([]byte(query))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) signedRequest(method, path string, params url.Values) ([]byte, int, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("timestamp", strconv.FormatInt(c.timestamp(), 10))
	params.Set("recvWindow", "10000")
	params.Set("signature", c.sign(params))

	var req *http.Request
	var err error

	if method == "GET" {
		req, err = http.NewRequest(method, c.baseURL+path+"?"+params.Encode(), nil)
	} else {
		req, err = http.NewRequest(method, c.baseURL+path, strings.NewReader(params.Encode()))
		if req != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}

	return body, resp.StatusCode, nil
}

func (c *Client) GetPrice(symbol string) (decimal.Decimal, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/fapi/v1/ticker/price?symbol=" + symbol)
	if err != nil {
		return decimal.Zero, fmt.Errorf("price request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Price string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return decimal.Zero, fmt.Errorf("price decode: %w", err)
	}

	price, err := decimal.NewFromString(result.Price)
	if err != nil {
		return decimal.Zero, fmt.Errorf("price parse: %w", err)
	}
	return price, nil
}

func (c *Client) LoadSymbolInfo(symbol string) (SymbolInfo, error) {
	c.symbolInfoMu.RLock()
	if info, ok := c.symbolInfo[symbol]; ok {
		c.symbolInfoMu.RUnlock()
		return info, nil
	}
	c.symbolInfoMu.RUnlock()

	resp, err := c.httpClient.Get(c.baseURL + "/fapi/v1/exchangeInfo")
	if err != nil {
		return SymbolInfo{}, fmt.Errorf("exchangeInfo: %w", err)
	}
	defer resp.Body.Close()

	var exInfo struct {
		Symbols []struct {
			Symbol         string `json:"symbol"`
			PricePrecision int    `json:"pricePrecision"`
			QuantityPrecision int `json:"quantityPrecision"`
			Filters        []struct {
				FilterType string `json:"filterType"`
				StepSize   string `json:"stepSize"`
				MinQty     string `json:"minQty"`
				TickSize   string `json:"tickSize"`
			} `json:"filters"`
		} `json:"symbols"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&exInfo); err != nil {
		return SymbolInfo{}, fmt.Errorf("exchangeInfo decode: %w", err)
	}

	c.symbolInfoMu.Lock()
	defer c.symbolInfoMu.Unlock()

	for _, s := range exInfo.Symbols {
		si := SymbolInfo{
			PricePrecision: s.PricePrecision,
			QtyPrecision:   s.QuantityPrecision,
		}
		for _, f := range s.Filters {
			switch f.FilterType {
			case "LOT_SIZE":
				si.StepSize, _ = decimal.NewFromString(f.StepSize)
				si.MinQty, _ = decimal.NewFromString(f.MinQty)
			case "PRICE_FILTER":
				si.TickSize, _ = decimal.NewFromString(f.TickSize)
			}
		}
		c.symbolInfo[s.Symbol] = si
	}

	info, ok := c.symbolInfo[symbol]
	if !ok {
		return SymbolInfo{}, fmt.Errorf("symbol %s not found in exchangeInfo", symbol)
	}
	return info, nil
}

func (c *Client) RoundQty(symbol string, qty decimal.Decimal) (decimal.Decimal, error) {
	info, err := c.LoadSymbolInfo(symbol)
	if err != nil {
		return decimal.Zero, err
	}
	if info.StepSize.IsZero() {
		return qty.Round(int32(info.QtyPrecision)), nil
	}
	return qty.Div(info.StepSize).Floor().Mul(info.StepSize), nil
}

func (c *Client) SetLeverage(symbol string, leverage int) error {
	params := url.Values{
		"symbol":   {symbol},
		"leverage": {strconv.Itoa(leverage)},
	}
	body, code, err := c.signedRequest("POST", "/fapi/v1/leverage", params)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("set leverage %d for %s: status %d, body: %s", leverage, symbol, code, string(body))
	}
	return nil
}

func (c *Client) SetMarginType(symbol, marginType string) error {
	params := url.Values{
		"symbol":     {symbol},
		"marginType": {strings.ToUpper(marginType)},
	}
	body, code, err := c.signedRequest("POST", "/fapi/v1/marginType", params)
	if err != nil {
		return err
	}
	// -4046 means margin type already set, which is fine
	if code != 200 {
		var apiErr struct {
			Code int `json:"code"`
		}
		json.Unmarshal(body, &apiErr)
		if apiErr.Code == -4046 {
			return nil
		}
		return fmt.Errorf("set margin type %s for %s: status %d, body: %s", marginType, symbol, code, string(body))
	}
	return nil
}

func (c *Client) PlaceMarketOrder(symbol, side string, quantity decimal.Decimal, reduceOnly bool) (*OrderResponse, []byte, []byte, error) {
	params := url.Values{
		"symbol":   {symbol},
		"side":     {strings.ToUpper(side)},
		"type":     {"MARKET"},
		"quantity": {quantity.String()},
	}
	if reduceOnly {
		params.Set("reduceOnly", "true")
	}

	reqJSON, _ := json.Marshal(map[string]string{
		"symbol": symbol, "side": side, "type": "MARKET",
		"quantity": quantity.String(), "reduceOnly": strconv.FormatBool(reduceOnly),
	})

	body, code, err := c.signedRequest("POST", "/fapi/v1/order", params)
	if err != nil {
		return nil, reqJSON, nil, fmt.Errorf("order request: %w", err)
	}

	if code != 200 {
		return nil, reqJSON, body, fmt.Errorf("order failed: status %d, body: %s", code, string(body))
	}

	var order OrderResponse
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, reqJSON, body, fmt.Errorf("order decode: %w", err)
	}

	return &order, reqJSON, body, nil
}

func (c *Client) Ping() error {
	resp, err := c.httpClient.Get(c.baseURL + "/fapi/v1/ping")
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ping status: %d", resp.StatusCode)
	}
	return nil
}
