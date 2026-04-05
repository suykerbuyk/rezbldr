// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package vault

// Contact represents the profile/contact.md vault file.
type Contact struct {
	Name               string   `yaml:"name"`
	Email              string   `yaml:"email"`
	Phone              string   `yaml:"phone"`
	Location           string   `yaml:"location"`
	LinkedIn           string   `yaml:"linkedin"`
	GitHub             string   `yaml:"github"`
	Tagline            string   `yaml:"tagline"`
	Languages          []string `yaml:"languages"`
	InternationalTeams string   `yaml:"international_teams"`
}

// LoadContact loads a contact profile from a markdown file.
func LoadContact(path string) (*Contact, error) {
	var c Contact
	_, err := loadAndParse(path, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
