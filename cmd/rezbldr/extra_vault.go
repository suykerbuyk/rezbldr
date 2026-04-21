// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// extraVaultFlag is a flag.Value that accumulates repeated --extra-vault
// entries of the form name=path into a map.
type extraVaultFlag struct {
	values map[string]string
}

func newExtraVaultFlag() *extraVaultFlag {
	return &extraVaultFlag{values: map[string]string{}}
}

// String renders the flag's current entries as a comma-joined name=path list,
// sorted by name for deterministic output.
func (f *extraVaultFlag) String() string {
	if f == nil || len(f.values) == 0 {
		return ""
	}
	return strings.Join(f.sortedEntries(), ",")
}

// Set parses a single name=path entry and stores it. Later values for the
// same name overwrite earlier ones, matching typical flag-override behavior.
func (f *extraVaultFlag) Set(value string) error {
	name, path, ok := strings.Cut(value, "=")
	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)
	if !ok || name == "" || path == "" {
		return fmt.Errorf("extra-vault must be of the form name=path, got %q", value)
	}
	expanded, err := expandHomePath(path)
	if err != nil {
		return fmt.Errorf("expanding %s: %w", path, err)
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[name] = expanded
	return nil
}

// sortedEntries returns name=path strings in name order.
func (f *extraVaultFlag) sortedEntries() []string {
	names := make([]string, 0, len(f.values))
	for name := range f.values {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, name+"="+f.values[name])
	}
	return out
}

// parseExtraVaultsEnv parses REZBLDR_EXTRA_VAULTS="name=path,name2=path2".
// Empty input returns nil, nil. Whitespace around entries and components is
// trimmed.
func parseExtraVaultsEnv(env string) (map[string]string, error) {
	env = strings.TrimSpace(env)
	if env == "" {
		return nil, nil
	}
	result := map[string]string{}
	for _, entry := range strings.Split(env, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		name, path, ok := strings.Cut(entry, "=")
		name = strings.TrimSpace(name)
		path = strings.TrimSpace(path)
		if !ok || name == "" || path == "" {
			return nil, fmt.Errorf("invalid REZBLDR_EXTRA_VAULTS entry %q (expected name=path)", entry)
		}
		expanded, err := expandHomePath(path)
		if err != nil {
			return nil, err
		}
		result[name] = expanded
	}
	return result, nil
}

// expandHomePath expands a leading ~/ to the user's home directory and
// cleans the result.
func expandHomePath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Clean(path), nil
}

// mergeExtraVaults returns the union of flag-supplied and env-supplied
// extra-vault entries. Flag entries win on conflict.
func mergeExtraVaults(flag, env map[string]string) map[string]string {
	out := map[string]string{}
	for name, path := range env {
		out[name] = path
	}
	for name, path := range flag {
		out[name] = path
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// resolveWrapRepo returns the repo directory a wrap call should target.
// vaultName == "" selects the default VaultPath. A named vault must be
// registered in cfg.ExtraVaults or an error is returned listing the
// registered names.
func resolveWrapRepo(cfg *Config, vaultName string) (string, error) {
	if vaultName == "" {
		return cfg.VaultPath, nil
	}
	if path, ok := cfg.ExtraVaults[vaultName]; ok {
		return path, nil
	}
	names := make([]string, 0, len(cfg.ExtraVaults))
	for name := range cfg.ExtraVaults {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", fmt.Errorf("unknown vault %q; no extra vaults are registered (pass --extra-vault name=path to rezbldr serve/setup)", vaultName)
	}
	return "", fmt.Errorf("unknown vault %q; registered extras: %s", vaultName, strings.Join(names, ", "))
}
