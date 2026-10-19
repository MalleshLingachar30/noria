// Package config owns private local configuration, never native user profiles.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/amxv/noria/internal/model"
)

const DefaultListen = "127.0.0.1:43119"

type Config struct {
	Listen   string          `json:"listen"`
	Accounts []model.Account `json:"accounts"`
}

func Home() (string, error) {
	h := os.Getenv("NORIA_HOME")
	if h == "" {
		u, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		h = filepath.Join(u, ".noria")
	}
	return filepath.Abs(h)
}

func PrivateDir(path string) error {
	if st, e := os.Lstat(path); e == nil && st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink directory %s", path)
	}
	if e := os.MkdirAll(path, 0700); e != nil {
		return e
	}
	return os.Chmod(path, 0700)
}

func Write(path string, data []byte) error {
	if e := PrivateDir(filepath.Dir(path)); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".noria-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(data)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}

func Init(home string) error {
	if e := PrivateDir(home); e != nil {
		return e
	}
	unlock, e := Lock(home, "config")
	if e != nil {
		return e
	}
	defer unlock()
	path := filepath.Join(home, "config.json")
	if _, e = os.Stat(path); errors.Is(e, os.ErrNotExist) {
		if e = Save(home, Config{Listen: DefaultListen, Accounts: []model.Account{}}); e != nil {
			return e
		}
	} else if e != nil {
		return e
	}
	secretPath := filepath.Join(home, "secret")
	if _, e = os.Stat(secretPath); errors.Is(e, os.ErrNotExist) {
		var b [32]byte
		if _, e = rand.Read(b[:]); e != nil {
			return e
		}
		return Write(secretPath, []byte(hex.EncodeToString(b[:])))
	} else {
		return e
	}
}

func Load(home string) (Config, error) {
	var c Config
	b, e := os.ReadFile(filepath.Join(home, "config.json"))
	if e != nil {
		return c, fmt.Errorf("read configuration (run noria init): %w", e)
	}
	if e = json.Unmarshal(b, &c); e != nil {
		return c, fmt.Errorf("invalid configuration: %w", e)
	}
	if e = ValidateListen(c.Listen); e != nil {
		return c, e
	}
	ids := map[string]bool{}
	for _, a := range c.Accounts {
		if e = ValidateID(a.ID); e != nil {
			return c, e
		}
		if ids[a.ID] {
			return c, fmt.Errorf("duplicate account %q", a.ID)
		}
		ids[a.ID] = true
		if a.Provider != "codex" && a.Provider != "claude" {
			return c, fmt.Errorf("invalid provider for %s", a.ID)
		}
		expected := filepath.Join(home, "accounts", a.ID)
		if a.Profile != expected {
			return c, fmt.Errorf("account %s profile must be %s", a.ID, expected)
		}
	}
	return c, nil
}

func Save(home string, c Config) error {
	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	return Write(filepath.Join(home, "config.json"), append(b, '\n'))
}
func Secret(home string) (string, error) {
	b, e := os.ReadFile(filepath.Join(home, "secret"))
	if e != nil {
		return "", e
	}
	if len(b) != 64 {
		return "", fmt.Errorf("invalid local secret; restore a 64-character secret")
	}
	return string(b), nil
}

var validID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func ValidateID(id string) error {
	if !validID.MatchString(id) {
		return fmt.Errorf("account ID must be 1-64 letters, digits, underscores or hyphens")
	}
	return nil
}
func ValidateListen(addr string) error {
	h, _, e := net.SplitHostPort(addr)
	if e != nil {
		return fmt.Errorf("invalid listen address: %w", e)
	}
	ip := net.ParseIP(h)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("listen must use a numeric loopback address")
	}
	return nil
}

// Lock serializes short control operations across processes. Locks are never
// silently stolen; a crashed command leaves an explicit recovery instruction.
func Lock(home, name string) (func(), error) {
	if e := PrivateDir(home); e != nil {
		return nil, e
	}
	p := filepath.Join(home, name+".lock")
	deadline := time.Now().Add(3 * time.Second)
	for {
		f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e == nil {
			fmt.Fprintf(f, "pid=%d\n", os.Getpid())
			f.Close()
			return func() { _ = os.Remove(p) }, nil
		}
		if !errors.Is(e, os.ErrExist) {
			return nil, e
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%s is locked; if no noria process owns it, remove %s", name, p)
		}
		timer := time.NewTimer(25 * time.Millisecond)
		<-timer.C
	}
}
