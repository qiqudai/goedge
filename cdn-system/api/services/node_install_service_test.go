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

func TestSSHAddressSupportsIPv6(t *testing.T) {
	if got := sshAddress("2001:db8::1", 22); got != "[2001:db8::1]:22" {
		t.Fatalf("sshAddress IPv6 = %q", got)
	}
	if got := sshAddress("192.0.2.10", 2222); got != "192.0.2.10:2222" {
		t.Fatalf("sshAddress IPv4 = %q", got)
	}
}
