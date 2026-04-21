// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtraVaultFlag_Set(t *testing.T) {
	f := newExtraVaultFlag()
	if err := f.Set("vibe=/tmp/a"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := f.Set("palace=/tmp/b"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := f.values["vibe"]; got != "/tmp/a" {
		t.Errorf("vibe = %q, want /tmp/a", got)
	}
	if got := f.values["palace"]; got != "/tmp/b" {
		t.Errorf("palace = %q, want /tmp/b", got)
	}
}

func TestExtraVaultFlag_SetOverwritesSameName(t *testing.T) {
	f := newExtraVaultFlag()
	_ = f.Set("vibe=/tmp/a")
	_ = f.Set("vibe=/tmp/b")
	if got := f.values["vibe"]; got != "/tmp/b" {
		t.Errorf("vibe = %q, want /tmp/b (later Set should win)", got)
	}
}

func TestExtraVaultFlag_SetRejectsInvalid(t *testing.T) {
	cases := []string{
		"",
		"novalue",
		"=onlypath",
		"onlyname=",
		"   =   ",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			f := newExtraVaultFlag()
			if err := f.Set(c); err == nil {
				t.Errorf("Set(%q) = nil, want error", c)
			}
		})
	}
}

func TestExtraVaultFlag_ExpandsHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	f := newExtraVaultFlag()
	if err := f.Set("vibe=~/obsidian/VibeVault"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	want := filepath.Join(home, "obsidian", "VibeVault")
	if got := f.values["vibe"]; got != want {
		t.Errorf("vibe = %q, want %q", got, want)
	}
}

func TestExtraVaultFlag_SortedEntries(t *testing.T) {
	f := newExtraVaultFlag()
	_ = f.Set("zebra=/z")
	_ = f.Set("apple=/a")
	_ = f.Set("mango=/m")
	got := f.sortedEntries()
	want := []string{"apple=/a", "mango=/m", "zebra=/z"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExtraVaultFlag_String(t *testing.T) {
	var nilFlag *extraVaultFlag
	if s := nilFlag.String(); s != "" {
		t.Errorf("nil.String() = %q, want empty", s)
	}
	empty := newExtraVaultFlag()
	if s := empty.String(); s != "" {
		t.Errorf("empty.String() = %q, want empty", s)
	}
	f := newExtraVaultFlag()
	_ = f.Set("b=/b")
	_ = f.Set("a=/a")
	if got := f.String(); got != "a=/a,b=/b" {
		t.Errorf("String() = %q, want a=/a,b=/b", got)
	}
}

func TestParseExtraVaultsEnv(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	tests := []struct {
		name    string
		env     string
		want    map[string]string
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"whitespace only", "   ", nil, false},
		{"single", "vibe=/tmp/v", map[string]string{"vibe": "/tmp/v"}, false},
		{
			"multiple with whitespace",
			" vibe=/tmp/v , palace=/tmp/p ",
			map[string]string{"vibe": "/tmp/v", "palace": "/tmp/p"},
			false,
		},
		{
			"home-expansion",
			"vibe=~/obsidian/VibeVault",
			map[string]string{"vibe": filepath.Join(home, "obsidian", "VibeVault")},
			false,
		},
		{"missing equals", "vibe/tmp/v", nil, true},
		{"missing name", "=/tmp/v", nil, true},
		{"missing path", "vibe=", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExtraVaultsEnv(tt.env)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d (got %v)", len(got), len(tt.want), got)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("%s = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestMergeExtraVaults(t *testing.T) {
	tests := []struct {
		name string
		flag map[string]string
		env  map[string]string
		want map[string]string
	}{
		{"both nil", nil, nil, nil},
		{"flag only", map[string]string{"a": "/a"}, nil, map[string]string{"a": "/a"}},
		{"env only", nil, map[string]string{"a": "/a"}, map[string]string{"a": "/a"}},
		{
			"disjoint union",
			map[string]string{"a": "/a"},
			map[string]string{"b": "/b"},
			map[string]string{"a": "/a", "b": "/b"},
		},
		{
			"flag wins on conflict",
			map[string]string{"a": "/flag"},
			map[string]string{"a": "/env"},
			map[string]string{"a": "/flag"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeExtraVaults(tt.flag, tt.env)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d (got %v)", len(got), len(tt.want), got)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("%s = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestResolveWrapRepo(t *testing.T) {
	cfg := &Config{
		VaultPath:   "/default",
		ExtraVaults: map[string]string{"vibe": "/vibe", "palace": "/palace"},
	}

	tests := []struct {
		name      string
		vault     string
		want      string
		wantErr   bool
		errSubstr string
	}{
		{"empty selects default", "", "/default", false, ""},
		{"named hit", "vibe", "/vibe", false, ""},
		{"named hit palace", "palace", "/palace", false, ""},
		{"unknown lists registered", "nope", "", true, "registered extras: palace, vibe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveWrapRepo(cfg, tt.vault)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q missing substring %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveWrapRepo_NoExtras(t *testing.T) {
	cfg := &Config{VaultPath: "/default"}
	_, err := resolveWrapRepo(cfg, "vibe")
	if err == nil {
		t.Fatal("expected error when no extras registered")
	}
	if !strings.Contains(err.Error(), "no extra vaults are registered") {
		t.Errorf("error %q missing guidance", err.Error())
	}
}
