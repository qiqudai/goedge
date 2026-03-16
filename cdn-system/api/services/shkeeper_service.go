package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultShkeeperTimeoutSec = 12
)

type ShkeeperSettings struct {
	Enable         bool
	BaseURL        string
	APIKey         string
	CallbackAPIKey string
	CryptoName     string
	Fiat           string
	CallbackURL    string
	TimeoutSec     int
}

type ShkeeperInvoiceCreateRequest struct {
	ExternalID  string `json:"external_id"`
	Fiat        string `json:"fiat"`
	Amount      string `json:"amount"`
	CallbackURL string `json:"callback_url"`
}

type ShkeeperInvoiceCreateResponse struct {
	Amount           string `json:"amount"`
	DisplayName      string `json:"display_name"`
	ExchangeRate     string `json:"exchange_rate"`
	ID               int64  `json:"id"`
	RecalculateAfter int64  `json:"recalculate_after"`
	Status           string `json:"status"`
	Wallet           string `json:"wallet"`
	Message          string `json:"message"`
}

type ShkeeperQuoteRequest struct {
	Fiat   string `json:"fiat"`
	Amount string `json:"amount"`
}

type ShkeeperQuoteResponse struct {
	CryptoAmount string `json:"crypto_amount"`
	ExchangeRate string `json:"exchange_rate"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

type ShkeeperCallbackTx struct {
	TxID         string      `json:"txid"`
	Date         string      `json:"date"`
	AmountCrypto interface{} `json:"amount_crypto"`
	AmountFiat   interface{} `json:"amount_fiat"`
	Trigger      bool        `json:"trigger"`
	Crypto       string      `json:"crypto"`
}

type ShkeeperCallbackPayload struct {
	ExternalID    string               `json:"external_id"`
	Crypto        string               `json:"crypto"`
	Addr          string               `json:"addr"`
	Fiat          string               `json:"fiat"`
	BalanceFiat   interface{}          `json:"balance_fiat"`
	BalanceCrypto interface{}          `json:"balance_crypto"`
	Paid          bool                 `json:"paid"`
	Status        string               `json:"status"`
	Transactions  []ShkeeperCallbackTx `json:"transactions"`
	FeePercent    interface{}          `json:"fee_percent"`
	OverpaidFiat  interface{}          `json:"overpaid_fiat"`
}

func LoadShkeeperSettings() (*ShkeeperSettings, error) {
	cfg, err := LoadSystemConfig()
	if err != nil {
		return nil, err
	}
	settings := &ShkeeperSettings{
		Enable:         ParseBoolFlag(cfg["pay_shkeeper_enable"]),
		BaseURL:        strings.TrimRight(strings.TrimSpace(cfg["pay_shkeeper_base_url"]), "/"),
		APIKey:         strings.TrimSpace(cfg["pay_shkeeper_api_key"]),
		CallbackAPIKey: strings.TrimSpace(cfg["pay_shkeeper_callback_api_key"]),
		CryptoName:     strings.TrimSpace(cfg["pay_shkeeper_crypto"]),
		Fiat:           strings.ToUpper(strings.TrimSpace(cfg["pay_shkeeper_fiat"])),
		CallbackURL:    strings.TrimSpace(cfg["pay_shkeeper_callback_url"]),
		TimeoutSec:     parsePositiveInt(cfg["pay_shkeeper_timeout_sec"], defaultShkeeperTimeoutSec),
	}
	if settings.CryptoName == "" {
		settings.CryptoName = "TRX-USDT"
	}
	if settings.Fiat == "" {
		settings.Fiat = "USD"
	}
	if settings.TimeoutSec <= 0 {
		settings.TimeoutSec = defaultShkeeperTimeoutSec
	}
	return settings, nil
}

func (s *ShkeeperSettings) ValidateForCreateInvoice() error {
	if s == nil {
		return errors.New("shkeeper settings is nil")
	}
	if !s.Enable {
		return errors.New("shkeeper is disabled")
	}
	if strings.TrimSpace(s.BaseURL) == "" {
		return errors.New("pay_shkeeper_base_url is empty")
	}
	if strings.TrimSpace(s.APIKey) == "" {
		return errors.New("pay_shkeeper_api_key is empty")
	}
	if strings.TrimSpace(s.CryptoName) == "" {
		return errors.New("pay_shkeeper_crypto is empty")
	}
	return nil
}

func (s *ShkeeperSettings) IsValidCallbackKey(headerValue string) bool {
	expected := strings.TrimSpace(s.CallbackAPIKey)
	if expected == "" {
		return false
	}
	return strings.TrimSpace(headerValue) == expected
}

func ShkeeperCreateInvoice(settings *ShkeeperSettings, req ShkeeperInvoiceCreateRequest) (*ShkeeperInvoiceCreateResponse, error) {
	if err := settings.ValidateForCreateInvoice(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ExternalID) == "" {
		return nil, errors.New("external_id is required")
	}
	if strings.TrimSpace(req.Amount) == "" {
		return nil, errors.New("amount is required")
	}
	if strings.TrimSpace(req.CallbackURL) == "" {
		return nil, errors.New("callback_url is required")
	}
	if strings.TrimSpace(req.Fiat) == "" {
		req.Fiat = settings.Fiat
	}
	endpoint := fmt.Sprintf("%s/api/v1/%s/payment_request", settings.BaseURL, reqPathEscape(settings.CryptoName))
	payloadBytes, _ := json.Marshal(req)
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Shkeeper-Api-Key", settings.APIKey)

	client := &http.Client{Timeout: time.Duration(settings.TimeoutSec) * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	var resp ShkeeperInvoiceCreateResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, err
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		msg := strings.TrimSpace(resp.Message)
		if msg == "" {
			msg = fmt.Sprintf("shkeeper create invoice failed, status=%d", httpResp.StatusCode)
		}
		return nil, errors.New(msg)
	}
	if !strings.EqualFold(strings.TrimSpace(resp.Status), "success") {
		msg := strings.TrimSpace(resp.Message)
		if msg == "" {
			msg = "shkeeper create invoice returns non-success status"
		}
		return nil, errors.New(msg)
	}
	return &resp, nil
}

func ShkeeperQuote(settings *ShkeeperSettings, req ShkeeperQuoteRequest) (*ShkeeperQuoteResponse, error) {
	if err := settings.ValidateForCreateInvoice(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Fiat) == "" {
		req.Fiat = settings.Fiat
	}
	if strings.TrimSpace(req.Amount) == "" {
		return nil, errors.New("amount is required")
	}
	endpoint := fmt.Sprintf("%s/api/v1/%s/quote", settings.BaseURL, reqPathEscape(settings.CryptoName))
	payloadBytes, _ := json.Marshal(req)
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Shkeeper-Api-Key", settings.APIKey)

	client := &http.Client{Timeout: time.Duration(settings.TimeoutSec) * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	var resp ShkeeperQuoteResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, err
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		msg := strings.TrimSpace(resp.Message)
		if msg == "" {
			msg = fmt.Sprintf("shkeeper quote failed, status=%d", httpResp.StatusCode)
		}
		return nil, errors.New(msg)
	}
	if !strings.EqualFold(strings.TrimSpace(resp.Status), "success") {
		msg := strings.TrimSpace(resp.Message)
		if msg == "" {
			msg = "shkeeper quote returns non-success status"
		}
		return nil, errors.New(msg)
	}
	return &resp, nil
}

func parsePositiveInt(raw string, fallback int) int {
	val, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || val <= 0 {
		return fallback
	}
	return val
}

func reqPathEscape(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}
