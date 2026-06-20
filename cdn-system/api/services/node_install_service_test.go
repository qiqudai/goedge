package services

import (
	"testing"

	"cdn-api/models"
)

func TestBuildInstallConfigDefaultsSSHUserToRoot(t *testing.T) {
	cfg, err := buildInstallConfig(&models.Node{
		ID:          1,
		IP:          "192.0.2.10",
		Token:       "node-token",
		SSHPort:     22,
		SSHAuthType: "password",
		SSHPassword: "secret",
	}, "https://api.example.com")
	if err != nil {
		t.Fatalf("buildInstallConfig returned error: %v", err)
	}
	if cfg.User != defaultSSHUser {
		t.Fatalf("SSH user = %q, want %q", cfg.User, defaultSSHUser)
	}
}
