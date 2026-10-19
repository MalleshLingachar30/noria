// Package integration installs reversible, user-approved native client routing.
package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/amxv/noria/internal/config"
	"github.com/pelletier/go-toml/v2"
)

type Field struct {
	Path      []string `json:"path"`
	Before    any      `json:"before"`
	Existed   bool     `json:"existed"`
	Installed any      `json:"installed"`
}
type Entry struct {
	Provider   string  `json:"provider"`
	Path       string  `json:"path"`
	Active     bool    `json:"active"`
	Backup     string  `json:"backup"`
	Fields     []Field `json:"fields,omitempty"`
	RootLine   string  `json:"root_line,omitempty"`
	RootBlock  string  `json:"root_block,omitempty"`
	Definition string  `json:"definition,omitempty"`
	Created    bool    `json:"created"`
}
type Store struct {
	Entries []Entry `json:"entries"`
}
type Target struct {
	Provider   string `json:"provider"`
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	Active     bool   `json:"active"`
	Registered bool   `json:"registered"`
}

func Load(home string) (Store, error) {
	s := Store{Entries: []Entry{}}
	b, e := os.ReadFile(filepath.Join(home, "integrations.json"))
	if errors.Is(e, os.ErrNotExist) {
		return s, nil
	}
	if e != nil {
		return s, e
	}
	e = json.Unmarshal(b, &s)
	return s, e
}
func save(home string, s Store) error {
	b, e := json.MarshalIndent(s, "", "  ")
	if e != nil {
		return e
	}
	return config.Write(filepath.Join(home, "integrations.json"), b)
}

func Discover(home string) ([]Target, error) {
	s, e := Load(home)
	if e != nil {
		return nil, e
	}
	u, e := os.UserHomeDir()
	if e != nil {
		return nil, e
	}
	targets := []Target{}
	seen := map[string]bool{}
	add := func(provider, root string, required bool) {
		if root == "" {
			return
		}
		name := "config.toml"
		if provider == "claude" {
			name = "settings.json"
		}
		p, _ := filepath.Abs(filepath.Join(root, name))
		if seen[p] {
			return
		}
		_, err := os.Stat(root)
		if !required && err != nil {
			return
		}
		seen[p] = true
		t := Target{Provider: provider, Path: p}
		_, err = os.Stat(p)
		t.Exists = err == nil
		for _, v := range s.Entries {
			if v.Path == p {
				t.Active = v.Active
				t.Registered = true
			}
		}
		targets = append(targets, t)
	}
	for _, provider := range []string{"codex", "claude"} {
		env := "CODEX_HOME"
		if provider == "claude" {
			env = "CLAUDE_CONFIG_DIR"
		}
		root := os.Getenv(env)
		if root == "" {
			root = filepath.Join(u, "."+provider)
		}
		add(provider, root, true)
		add(provider, filepath.Join(u, "."+provider), false)
		add(provider, filepath.Join(u, ".gg", provider), false)
	}
	for _, v := range s.Entries {
		if !seen[v.Path] {
			targets = append(targets, Target{Provider: v.Provider, Path: v.Path, Exists: true, Active: v.Active, Registered: true})
		}
	}
	return targets, nil
}

// Preview contains paths and non-secret configuration only.
func Preview(home, provider, root, executable, address string) (map[string]any, error) {
	path, e := TargetPath(provider, root)
	if e != nil {
		return nil, e
	}
	before, e := read(path)
	if e != nil {
		return nil, e
	}
	store, e := Load(home)
	if e != nil {
		return nil, e
	}
	for _, registered := range store.Entries {
		if registered.Path == path && registered.Active {
			return map[string]any{"provider": provider, "path": path, "changes": changes(provider, home, executable, address), "note": "Already registered. Removing restores only Noria-managed routing fields; restart existing clients after changing routes."}, nil
		}
	}
	entry := Entry{Provider: provider, Path: path}
	after, e := activate(home, executable, address, before, &entry)
	if e != nil {
		return nil, e
	}
	_ = after
	return map[string]any{"provider": provider, "path": path, "changes": changes(provider, home, executable, address), "note": "Backs up this file; preserves other settings and native login. Stop restores direct settings. Existing clients may need restart."}, nil
}
func TargetPath(provider, root string) (string, error) {
	if provider != "codex" && provider != "claude" {
		return "", errors.New("client must be codex or claude")
	}
	if root == "" {
		env := "CODEX_HOME"
		if provider == "claude" {
			env = "CLAUDE_CONFIG_DIR"
		}
		root = os.Getenv(env)
		if root == "" {
			u, e := os.UserHomeDir()
			if e != nil {
				return "", e
			}
			root = filepath.Join(u, "."+provider)
		}
	}
	if strings.HasPrefix(root, "~"+string(os.PathSeparator)) {
		u, _ := os.UserHomeDir()
		root = filepath.Join(u, root[2:])
	}
	name := "config.toml"
	if provider == "claude" {
		name = "settings.json"
	}
	return filepath.Abs(filepath.Join(root, name))
}
func read(path string) ([]byte, error) {
	info, e := os.Lstat(path)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("refusing nonregular settings file: %s", path)
	}
	if info.Size() > 8<<20 {
		return nil, errors.New("settings file too large")
	}
	return os.ReadFile(path)
}
func write(path string, b []byte) error {
	// Do not chmod an existing client directory; it belongs to the native client.
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	if info, e := os.Lstat(path); e == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("refusing nonregular settings file %s", path)
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".noria-settings-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	if ce := f.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}

// Install is explicit opt-in. Every affected file is backed up before mutation.
func Install(home, provider, root, executable, address string) error {
	unlock, e := config.Lock(home, "integrations")
	if e != nil {
		return e
	}
	defer unlock()
	s, e := Load(home)
	if e != nil {
		return e
	}
	path, e := TargetPath(provider, root)
	if e != nil {
		return e
	}
	for _, v := range s.Entries {
		if v.Path == path && v.Active {
			return Check(v)
		}
	}
	before, e := read(path)
	if e != nil {
		return e
	}
	entry := Entry{Provider: provider, Path: path, Created: before == nil}
	after, e := activate(home, executable, address, before, &entry)
	if e != nil {
		return e
	}
	sum := sha256.Sum256([]byte(path))
	entry.Backup = filepath.Join(home, "backups", hex.EncodeToString(sum[:8])+".original")
	if _, e := os.Stat(entry.Backup); errors.Is(e, os.ErrNotExist) {
		if e = config.Write(entry.Backup, before); e != nil {
			return e
		}
	}
	old := append([]Entry(nil), s.Entries...)
	found := false
	for i, v := range s.Entries {
		if v.Path == path {
			s.Entries[i] = entry
			found = true
		}
	}
	if !found {
		s.Entries = append(s.Entries, entry)
	}
	// Store restoration data first. Recovery can recognize the installed fields
	// after an interrupted write; never leave an untracked proxy configuration.
	if e = save(home, s); e != nil {
		return e
	}
	if e = write(path, after); e != nil {
		_ = save(home, Store{Entries: old})
		return e
	}
	return nil
}
func SetActive(home, executable, address string, active bool) error {
	unlock, e := config.Lock(home, "integrations")
	if e != nil {
		return e
	}
	defer unlock()
	s, e := Load(home)
	if e != nil {
		return e
	}
	before := make(map[string][]byte)
	after := make(map[string][]byte)
	for i := range s.Entries {
		entry := &s.Entries[i]
		if entry.Active == active {
			continue
		}
		b, e := read(entry.Path)
		if e != nil {
			return e
		}
		before[entry.Path] = b
		if active {
			after[entry.Path], e = activate(home, executable, address, b, entry)
		} else {
			after[entry.Path], e = deactivate(b, entry)
		}
		if e != nil {
			return fmt.Errorf("%s: %w", entry.Path, e)
		}
	}
	written := []string{}
	rollback := func() {
		for _, p := range written {
			if before[p] == nil {
				_ = os.Remove(p)
			} else {
				_ = write(p, before[p])
			}
		}
	}
	for p, b := range after {
		if e = write(p, b); e != nil {
			rollback()
			return e
		}
		written = append(written, p)
	}
	if e = save(home, s); e != nil {
		rollback()
		return e
	}
	return nil
}
func Remove(home, provider, root string) error {
	unlock, e := config.Lock(home, "integrations")
	if e != nil {
		return e
	}
	defer unlock()
	path, e := TargetPath(provider, root)
	if e != nil {
		return e
	}
	s, e := Load(home)
	if e != nil {
		return e
	}
	for i := range s.Entries {
		v := &s.Entries[i]
		if v.Path != path {
			continue
		}
		b, e := read(path)
		if e != nil {
			return e
		}
		if v.Active {
			after, e := deactivate(b, v)
			if e != nil {
				return e
			}
			if e = write(path, after); e != nil {
				return e
			}
		}
		s.Entries = append(s.Entries[:i], s.Entries[i+1:]...)
		return save(home, s)
	}
	return nil
}
func Check(v Entry) error {
	b, e := read(v.Path)
	if e != nil {
		return e
	}
	if !v.Active {
		return nil
	}
	_, e = deactivate(b, &v)
	return e
}
func changes(provider, home, exe, address string) map[string]any {
	if provider == "codex" {
		return map[string]any{"model_provider": "noria", "model_providers": map[string]any{"noria": map[string]any{"name": "Noria", "base_url": "http://" + address + "/codex/v1", "wire_api": "responses", "supports_websockets": true, "auth": map[string]any{"command": exe, "args": []string{"auth-token", "--home", home}}}}}
	}
	return map[string]any{"apiKeyHelper": helperCommand(exe, home), "env": map[string]any{"ANTHROPIC_BASE_URL": "http://" + address + "/claude", "ENABLE_TOOL_SEARCH": "true"}}
}
func helperCommand(exe, home string) string {
	if runtime.GOOS == "windows" {
		return `"` + exe + `" auth-token --home "` + home + `"`
	}
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	return q(exe) + " auth-token --home " + q(home)
}
func activate(home, exe, address string, b []byte, v *Entry) ([]byte, error) {
	if v.Provider == "codex" {
		return activateCodex(home, exe, address, b, v)
	}
	root := map[string]any{}
	if len(bytes.TrimSpace(b)) > 0 {
		if e := json.Unmarshal(b, &root); e != nil {
			return nil, fmt.Errorf("invalid settings JSON: %w", e)
		}
	}
	if root == nil {
		return nil, errors.New("settings must be a JSON object")
	}
	v.Fields = nil
	for _, path := range [][]string{{"apiKeyHelper"}, {"env", "ANTHROPIC_BASE_URL"}, {"env", "ENABLE_TOOL_SEARCH"}} {
		installed, _ := get(changes(v.Provider, home, exe, address), path)
		previous, existed := get(root, path)
		v.Fields = append(v.Fields, Field{Path: path, Before: previous, Existed: existed, Installed: installed})
		if e := put(root, path, installed, false); e != nil {
			return nil, e
		}
	}
	v.Active = true
	out, e := json.MarshalIndent(root, "", "  ")
	return append(out, '\n'), e
}
func deactivate(b []byte, v *Entry) ([]byte, error) {
	if v.Provider == "codex" {
		source := string(b)
		for _, block := range []string{v.RootBlock, v.Definition} {
			if block == "" || strings.Count(source, block) != 1 {
				return nil, errors.New("managed routing was edited externally; refusing to overwrite it. Restore the Noria fields or remove integration manually")
			}
			source = strings.Replace(source, block, "", 1)
		}
		source = v.RootLine + source
		var parsed map[string]any
		if e := toml.Unmarshal([]byte(source), &parsed); e != nil {
			return nil, e
		}
		v.Active = false
		return []byte(source), nil
	}
	root := map[string]any{}
	if e := json.Unmarshal(b, &root); e != nil {
		return nil, e
	}
	for _, f := range v.Fields {
		now, exists := get(root, f.Path)
		if !exists || !equal(now, f.Installed) {
			return nil, fmt.Errorf("managed %s was edited externally; refusing to overwrite", strings.Join(f.Path, "."))
		}
	}
	for _, f := range v.Fields {
		if e := put(root, f.Path, f.Before, !f.Existed); e != nil {
			return nil, e
		}
	}
	if env, ok := root["env"].(map[string]any); ok && len(env) == 0 {
		delete(root, "env")
	}
	out, e := json.MarshalIndent(root, "", "  ")
	v.Active = false
	return append(out, '\n'), e
}
func get(root map[string]any, path []string) (any, bool) {
	cur := root
	for _, p := range path[:len(path)-1] {
		next, ok := cur[p].(map[string]any)
		if !ok {
			return nil, false
		}
		cur = next
	}
	v, ok := cur[path[len(path)-1]]
	return v, ok
}
func put(root map[string]any, path []string, value any, remove bool) error {
	cur := root
	for _, p := range path[:len(path)-1] {
		next, ok := cur[p].(map[string]any)
		if !ok {
			if _, exists := cur[p]; exists {
				return fmt.Errorf("%s must be a table/object", p)
			}
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
	if remove {
		delete(cur, path[len(path)-1])
	} else {
		cur[path[len(path)-1]] = value
	}
	return nil
}
func equal(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return reflect.DeepEqual(x, y)
}

func activateCodex(home, exe, address string, b []byte, v *Entry) ([]byte, error) {
	var root map[string]any
	if e := toml.Unmarshal(b, &root); e != nil {
		return nil, fmt.Errorf("invalid TOML: %w", e)
	}
	if _, exists := get(root, []string{"model_providers", "noria"}); exists {
		return nil, errors.New("a noria provider already exists; use integration status or remove the previous manual definition first")
	}
	lines := strings.SplitAfter(string(b), "\n")
	v.RootLine = ""
	section := false
	found := false
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") {
			section = true
		}
		if section {
			continue
		}
		if strings.HasPrefix(trim, "model_provider") {
			key, _, ok := strings.Cut(trim, "=")
			if ok && strings.TrimSpace(key) == "model_provider" {
				v.RootLine = line
				lines[i] = ""
				found = true
				break
			}
		}
	}
	if _, exists := root["model_provider"]; exists && !found {
		return nil, errors.New("nonstandard model_provider declaration; edit it to a standalone unquoted key before integration")
	}
	v.RootBlock = "# >>> Noria routing (managed)\nmodel_provider = \"noria\"\n# <<< Noria routing\n"
	v.Definition = "\n# >>> Noria provider (managed)\n[model_providers.noria]\nname = \"Noria\"\nbase_url = " + strconv.Quote("http://"+address+"/codex/v1") + "\nwire_api = \"responses\"\nsupports_websockets = true\n[model_providers.noria.auth]\ncommand = " + strconv.Quote(exe) + "\nargs = [\"auth-token\", \"--home\", " + strconv.Quote(home) + "]\n# <<< Noria provider\n"
	source := v.RootBlock + strings.Join(lines, "") + v.Definition
	if e := toml.Unmarshal([]byte(source), &root); e != nil {
		return nil, e
	}
	v.Active = true
	return []byte(source), nil
}
func SortedTargets(t []Target) {
	sort.Slice(t, func(i, j int) bool {
		if t[i].Provider != t[j].Provider {
			return t[i].Provider < t[j].Provider
		}
		return t[i].Path < t[j].Path
	})
}

// Views is a presentation contract for the CLI/menu bar, not a second state store.
type View struct {
	Provider string `json:"provider"`
	Path     string `json:"path"`
	Active   bool   `json:"active"`
	Healthy  bool   `json:"healthy"`
	Error    string `json:"error,omitempty"`
}

func Views(home string) ([]View, error) {
	s, e := Load(home)
	if e != nil {
		return nil, e
	}
	out := []View{}
	for _, r := range s.Entries {
		v := View{Provider: r.Provider, Path: r.Path, Active: r.Active, Healthy: true}
		if e := Check(r); e != nil {
			v.Healthy = false
			v.Error = e.Error()
		}
		out = append(out, v)
	}
	return out, nil
}
func DefaultRoot(provider string) (string, error) {
	p, e := TargetPath(provider, "")
	if e != nil {
		return "", e
	}
	return filepath.Dir(p), nil
}
