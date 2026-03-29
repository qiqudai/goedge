package services

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/smtp"
	"strings"
)

type SMTPConfig struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"pwd"`
	UseSSL     bool   `json:"use_ssl"`
	ProxyIP    string `json:"proxy_ip"`
	ProxyPort  int    `json:"proxy_port"`
	ProxyUser  string `json:"proxy_user"`
	ProxyPwd   string `json:"proxy_pwd"`
	ProxyState bool   `json:"proxy_state"`
}

func LoadSMTPConfig() (*SMTPConfig, error) {
	cfg, err := LoadSystemConfig()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(cfg["smtp"])
	if raw == "" {
		return nil, errors.New("smtp config is empty")
	}
	var smtpCfg SMTPConfig
	if err := json.Unmarshal([]byte(raw), &smtpCfg); err != nil {
		return nil, err
	}
	if strings.TrimSpace(smtpCfg.IP) == "" || smtpCfg.Port == 0 {
		return nil, errors.New("smtp config incomplete")
	}
	return &smtpCfg, nil
}

func SendEmail(to, subject, htmlBody string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return errors.New("empty email")
	}
	cfg, err := LoadSMTPConfig()
	if err != nil {
		return err
	}
	host := strings.TrimSpace(cfg.IP)
	addr := fmt.Sprintf("%s:%d", host, cfg.Port)

	headers := make([]string, 0, 6)
	headers = append(headers, "From: "+cfg.User)
	headers = append(headers, "To: "+to)
	headers = append(headers, "Subject: "+subject)
	headers = append(headers, "MIME-Version: 1.0")
	headers = append(headers, "Content-Type: text/html; charset=UTF-8")
	message := strings.Join(headers, "\r\n") + "\r\n\r\n" + htmlBody

	var client *smtp.Client
	if cfg.UseSSL {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
		if err != nil {
			return err
		}
		client, err = smtp.NewClient(conn, host)
		if err != nil {
			return err
		}
	} else {
		client, err = smtp.Dial(addr)
		if err != nil {
			return err
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			_ = client.StartTLS(&tls.Config{ServerName: host})
		}
	}
	defer client.Close()

	if cfg.User != "" && cfg.Password != "" {
		auth := smtp.PlainAuth("", cfg.User, cfg.Password, host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(cfg.User); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}
