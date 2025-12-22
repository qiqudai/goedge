package models

import "time"

type DNSProvider struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`        // cloudflare, aliyun, dnspod, etc.
	Credentials string    `json:"credentials"` // JSON string storing keys/secrets
	CreatedAt   time.Time `json:"created_at"`
}
