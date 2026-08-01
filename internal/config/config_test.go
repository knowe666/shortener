package config

import (
	"flag"
	"os"
	"testing"
)

func TestNewConfigUsesAuthSecretFromEnv(t *testing.T) {
	oldArgs := os.Args
	oldAuthSecret := os.Getenv("AUTH_SECRET")
	oldCommandLine := flag.CommandLine

	defer func() {
		os.Args = oldArgs
		if oldAuthSecret == "" {
			_ = os.Unsetenv("AUTH_SECRET")
		} else {
			_ = os.Setenv("AUTH_SECRET", oldAuthSecret)
		}
		flag.CommandLine = oldCommandLine
	}()

	os.Args = []string{"shortener"}
	_ = os.Setenv("AUTH_SECRET", "test-secret-from-env")
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() error = %v", err)
	}

	if cfg.AuthSecret != "test-secret-from-env" {
		t.Fatalf("AuthSecret = %q, want %q", cfg.AuthSecret, "test-secret-from-env")
	}
}
