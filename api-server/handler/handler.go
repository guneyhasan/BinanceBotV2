package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"api-server/db"
	"api-server/models"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store              *db.Store
	telegramToken      string
	telegramSignalChat string
	telegramTradeChat  string
	binanceBaseURL     string
	httpClient         *http.Client
}

func New(store *db.Store) *Handler {
	return &Handler{
		store:              store,
		telegramToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
		telegramSignalChat: os.Getenv("TELEGRAM_SIGNAL_CHAT_ID"),
		telegramTradeChat:  os.Getenv("TELEGRAM_TRADE_CHAT_ID"),
		binanceBaseURL:     strings.TrimRight(envOrDefault("BINANCE_BASE_URL", "https://testnet.binancefuture.com"), "/"),
		httpClient:         &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-server", "time": time.Now().UTC()})
}

func (h *Handler) GetConfig(c *gin.Context) {
	cfg, err := h.store.GetConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *Handler) UpdateConfig(c *gin.Context) {
	var u models.ConfigUpdate
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if err := h.store.UpdateConfig(c.Request.Context(), u); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cfg, _ := h.store.GetConfig(c.Request.Context())
	c.JSON(http.StatusOK, cfg)
}

func (h *Handler) GetTrades(c *gin.Context) {
	coin := c.Query("coin")
	signalType := c.Query("signal_type")
	var active *bool
	if a := c.Query("active"); a != "" {
		v := a == "true"
		active = &v
	}
	trades, err := h.store.GetTrades(c.Request.Context(), coin, signalType, active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if trades == nil {
		trades = []models.Trade{}
	}
	c.JSON(http.StatusOK, trades)
}

func (h *Handler) GetActiveTrades(c *gin.Context) {
	active := true
	trades, err := h.store.GetTrades(c.Request.Context(), "", "", &active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if trades == nil {
		trades = []models.Trade{}
	}
	c.JSON(http.StatusOK, trades)
}

func (h *Handler) GetPnL(c *gin.Context) {
	coin := c.Query("coin")
	side := c.Query("side")
	records, err := h.store.GetPnLRecords(c.Request.Context(), coin, side)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if records == nil {
		records = []models.PnLRecord{}
	}
	c.JSON(http.StatusOK, records)
}

func (h *Handler) GetPnLSeries(c *gin.Context) {
	coin := c.Query("coin")
	side := c.Query("side")
	series, err := h.store.GetPnLSeries(c.Request.Context(), coin, side)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if series == nil {
		series = []models.PnLSeries{}
	}
	c.JSON(http.StatusOK, series)
}

func (h *Handler) GetPnLSummary(c *gin.Context) {
	coin := c.Query("coin")
	summaries, err := h.store.GetPnLSummary(c.Request.Context(), coin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if summaries == nil {
		summaries = []models.PnLSummary{}
	}
	c.JSON(http.StatusOK, summaries)
}

func (h *Handler) GetPnLCombined(c *gin.Context) {
	summaries, err := h.store.GetPnLSummary(c.Request.Context(), "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if summaries == nil {
		summaries = []models.PnLSummary{}
	}
	c.JSON(http.StatusOK, summaries)
}

func (h *Handler) GetUnrealizedPnL(c *gin.Context) {
	response, statusCode, err := h.buildUnrealizedPnL(c.Request.Context())
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) GetPnLLast24h(c *gin.Context) {
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)

	accountSummaries, err := h.store.GetPnL24hAccountSummaries(c.Request.Context(), from, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	coinSummaries, err := h.store.GetPnL24hCoinSummaries(c.Request.Context(), from, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	chart, err := h.store.GetPnL24hChartPoints(c.Request.Context(), from, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	unrealized, statusCode, err := h.buildUnrealizedPnL(c.Request.Context())
	if err != nil {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	accountByType := map[string]*models.PnL24hAccountSummary{}
	for _, accountType := range []string{"AL_ACCOUNT", "SAT_ACCOUNT"} {
		accountByType[accountType] = &models.PnL24hAccountSummary{
			AccountType: accountType,
			Label:       accountLabel(accountType),
		}
	}
	for i := range accountSummaries {
		sm := accountSummaries[i]
		sm.Label = accountLabel(sm.AccountType)
		sm.TotalNetPnL = sm.RealizedNetPnL
		accountByType[sm.AccountType] = &sm
	}

	coinByName := map[string]*models.PnL24hCoinSummary{}
	for i := range coinSummaries {
		sm := coinSummaries[i]
		sm.TotalNetPnL = sm.RealizedNetPnL
		coinByName[sm.Coin] = &sm
	}

	for _, item := range unrealized.Items {
		account, ok := accountByType[item.AccountType]
		if !ok {
			account = &models.PnL24hAccountSummary{
				AccountType: item.AccountType,
				Label:       accountLabel(item.AccountType),
			}
			accountByType[item.AccountType] = account
		}
		account.UnrealizedNetPnL += item.NetPnL
		account.TotalNetPnL = account.RealizedNetPnL + account.UnrealizedNetPnL

		coin, ok := coinByName[item.Coin]
		if !ok {
			coin = &models.PnL24hCoinSummary{Coin: item.Coin}
			coinByName[item.Coin] = coin
		}
		coin.UnrealizedNetPnL += item.NetPnL
		coin.TotalNetPnL = coin.RealizedNetPnL + coin.UnrealizedNetPnL
		if item.AccountType == "AL_ACCOUNT" {
			coin.ALNetPnL += item.NetPnL
		}
		if item.AccountType == "SAT_ACCOUNT" {
			coin.SATNetPnL += item.NetPnL
		}
	}

	accounts := make([]models.PnL24hAccountSummary, 0, len(accountByType))
	coins := make([]models.PnL24hCoinSummary, 0, len(coinByName))
	response := models.PnL24hResponse{
		From:                  from.Format(time.RFC3339),
		To:                    now.Format(time.RFC3339),
		UpdatedAt:             now.Format(time.RFC3339),
		Chart:                 chart,
		TotalUnrealizedNetPnL: unrealized.TotalNetPnL,
	}

	for _, accountType := range []string{"AL_ACCOUNT", "SAT_ACCOUNT"} {
		if account, ok := accountByType[accountType]; ok {
			response.TotalRealizedGross += account.RealizedGrossPnL
			response.TotalRealizedComm += account.RealizedComm
			response.TotalRealizedNetPnL += account.RealizedNetPnL
			accounts = append(accounts, *account)
			delete(accountByType, accountType)
		}
	}
	for _, account := range accountByType {
		response.TotalRealizedGross += account.RealizedGrossPnL
		response.TotalRealizedComm += account.RealizedComm
		response.TotalRealizedNetPnL += account.RealizedNetPnL
		accounts = append(accounts, *account)
	}
	response.TotalNetPnL = response.TotalRealizedNetPnL + response.TotalUnrealizedNetPnL

	for _, coin := range coinByName {
		coins = append(coins, *coin)
	}
	sort.Slice(coins, func(i, j int) bool {
		return coins[i].TotalNetPnL > coins[j].TotalNetPnL
	})

	response.Accounts = accounts
	response.Coins = coins
	response.Balances = h.buildAccountBalances()
	response.BalanceChart = buildBalanceChart(chart, response.Balances)

	c.JSON(http.StatusOK, response)
}

func (h *Handler) buildUnrealizedPnL(ctx context.Context) (*models.UnrealizedPnLResponse, int, error) {
	active := true
	trades, err := h.store.GetTrades(ctx, "", "", &active)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if trades == nil {
		trades = []models.Trade{}
	}

	cfg, err := h.store.GetConfig(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	priceCache := map[string]float64{}
	items := make([]models.UnrealizedPnLItem, 0, len(trades))
	response := models.UnrealizedPnLResponse{
		Items:     items,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, trade := range trades {
		currentPrice, ok := priceCache[trade.Coin]
		if !ok {
			currentPrice, err = h.fetchBinancePrice(trade.Coin)
			if err != nil {
				return nil, http.StatusBadGateway, err
			}
			priceCache[trade.Coin] = currentPrice
		}

		lev := float64(trade.Leverage)
		var grossPnL float64
		if trade.Side == "LONG" {
			grossPnL = (currentPrice - trade.EntryPrice) * trade.Quantity * lev
		} else {
			grossPnL = (trade.EntryPrice - currentPrice) * trade.Quantity * lev
		}

		openCommission := trade.Commission
		closeCommission := trade.Quantity * currentPrice * cfg.CommissionRate * lev
		totalCommission := openCommission + closeCommission
		netPnL := grossPnL - totalCommission

		item := models.UnrealizedPnLItem{
			TradeID:         trade.ID,
			Coin:            trade.Coin,
			SignalType:      trade.SignalType,
			Side:            trade.Side,
			AccountType:     trade.AccountType,
			Quantity:        trade.Quantity,
			EntryPrice:      trade.EntryPrice,
			CurrentPrice:    currentPrice,
			Leverage:        trade.Leverage,
			GrossPnL:        grossPnL,
			OpenCommission:  openCommission,
			CloseCommission: closeCommission,
			TotalCommission: totalCommission,
			NetPnL:          netPnL,
		}

		response.Items = append(response.Items, item)
		response.TotalGrossPnL += grossPnL
		response.TotalCommission += totalCommission
		response.TotalNetPnL += netPnL
	}

	return &response, http.StatusOK, nil
}

func (h *Handler) buildAccountBalances() []models.AccountBalance {
	accounts := []struct {
		accountType string
		apiKey      string
		apiSecret   string
	}{
		{accountType: "AL_ACCOUNT", apiKey: os.Getenv("BINANCE_AL_API_KEY"), apiSecret: os.Getenv("BINANCE_AL_API_SECRET")},
		{accountType: "SAT_ACCOUNT", apiKey: os.Getenv("BINANCE_SAT_API_KEY"), apiSecret: os.Getenv("BINANCE_SAT_API_SECRET")},
	}

	balances := make([]models.AccountBalance, 0, len(accounts))
	for _, account := range accounts {
		balance := models.AccountBalance{
			AccountType: account.accountType,
			Label:       accountLabel(account.accountType),
			Asset:       "USDT",
		}
		fetched, err := h.fetchAccountBalance(account.accountType, account.apiKey, account.apiSecret)
		if err != nil {
			balance.Error = err.Error()
			balances = append(balances, balance)
			continue
		}
		balances = append(balances, fetched)
	}
	return balances
}

func (h *Handler) fetchAccountBalance(accountType, apiKey, apiSecret string) (models.AccountBalance, error) {
	balance := models.AccountBalance{
		AccountType: accountType,
		Label:       accountLabel(accountType),
		Asset:       "USDT",
	}
	if apiKey == "" || apiSecret == "" {
		return balance, fmt.Errorf("%s Binance API anahtarlari eksik", accountLabel(accountType))
	}

	params := url.Values{}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", "10000")
	signature := signBinanceParams(params, apiSecret)
	params.Set("signature", signature)

	req, err := http.NewRequest(http.MethodGet, h.binanceBaseURL+"/fapi/v2/balance?"+params.Encode(), nil)
	if err != nil {
		return balance, err
	}
	req.Header.Set("X-MBX-APIKEY", apiKey)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return balance, fmt.Errorf("%s bakiye istegi basarisiz: %w", accountLabel(accountType), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return balance, fmt.Errorf("%s bakiye cevabi okunamadi: %w", accountLabel(accountType), err)
	}
	if resp.StatusCode != http.StatusOK {
		return balance, fmt.Errorf("%s bakiye istegi status %d: %s", accountLabel(accountType), resp.StatusCode, string(body))
	}

	var result []struct {
		Asset              string `json:"asset"`
		Balance            string `json:"balance"`
		AvailableBalance   string `json:"availableBalance"`
		CrossWalletBalance string `json:"crossWalletBalance"`
		CrossUnPnl         string `json:"crossUnPnl"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return balance, fmt.Errorf("%s bakiye cevabi parse edilemedi: %w", accountLabel(accountType), err)
	}

	for _, item := range result {
		if item.Asset != "USDT" {
			continue
		}
		balance.WalletBalance = parseBinanceFloat(item.Balance)
		balance.AvailableBalance = parseBinanceFloat(item.AvailableBalance)
		balance.CrossWalletBalance = parseBinanceFloat(item.CrossWalletBalance)
		balance.UnrealizedPnL = parseBinanceFloat(item.CrossUnPnl)
		return balance, nil
	}

	return balance, fmt.Errorf("%s USDT bakiyesi bulunamadi", accountLabel(accountType))
}

func buildBalanceChart(chart []models.PnL24hChartPoint, balances []models.AccountBalance) []models.BalanceChartPoint {
	alBalance := findBalance(balances, "AL_ACCOUNT")
	satBalance := findBalance(balances, "SAT_ACCOUNT")
	if len(chart) == 0 {
		return []models.BalanceChartPoint{}
	}

	last := chart[len(chart)-1]
	points := make([]models.BalanceChartPoint, 0, len(chart))
	for _, point := range chart {
		points = append(points, models.BalanceChartPoint{
			Time:       point.Time,
			ALBalance:  alBalance - last.ALCumulativeRealizedPnL + point.ALCumulativeRealizedPnL,
			SATBalance: satBalance - last.SATCumulativeRealizedPnL + point.SATCumulativeRealizedPnL,
		})
	}
	return points
}

func findBalance(balances []models.AccountBalance, accountType string) float64 {
	for _, balance := range balances {
		if balance.AccountType == accountType {
			return balance.WalletBalance
		}
	}
	return 0
}

func signBinanceParams(params url.Values, secret string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params.Get(key))
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.Join(parts, "&")))
	return hex.EncodeToString(mac.Sum(nil))
}

func parseBinanceFloat(value string) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func accountLabel(accountType string) string {
	switch accountType {
	case "AL_ACCOUNT":
		return "AL Hesabı"
	case "SAT_ACCOUNT":
		return "SAT Hesabı"
	default:
		return accountType
	}
}

func (h *Handler) fetchBinancePrice(symbol string) (float64, error) {
	endpoint := fmt.Sprintf("%s/fapi/v1/ticker/price?symbol=%s", h.binanceBaseURL, url.QueryEscape(symbol))
	resp, err := h.httpClient.Get(endpoint)
	if err != nil {
		return 0, fmt.Errorf("price request failed for %s: %w", symbol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("price request failed for %s: status %d", symbol, resp.StatusCode)
	}

	var body struct {
		Price string `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("price decode failed for %s: %w", symbol, err)
	}

	price, err := strconv.ParseFloat(body.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("price parse failed for %s: %w", symbol, err)
	}
	return price, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (h *Handler) GetWebhooks(c *gin.Context) {
	coin := c.Query("coin")
	signal := c.Query("signal")
	status := c.Query("status")
	webhooks, err := h.store.GetWebhooks(c.Request.Context(), coin, signal, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if webhooks == nil {
		webhooks = []models.WebhookLog{}
	}
	c.JSON(http.StatusOK, webhooks)
}

func (h *Handler) GetWebhookDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	webhook, err := h.store.GetWebhookDetail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if webhook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, webhook)
}

func (h *Handler) GetSystemHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "api-server",
		"time":    time.Now().UTC(),
	})
}

func (h *Handler) GetSystemStats(c *gin.Context) {
	stats, err := h.store.GetSystemStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
