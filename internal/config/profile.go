// Package config manages persisted connection profiles for softether-tui.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Mode identifies which vpncmd management mode a profile connects with.
type Mode string

const (
	ModeServer Mode = "server"
	ModeBridge Mode = "bridge"
	ModeClient Mode = "client"
)

// Profile is a saved connection target for vpncmd.
type Profile struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode Mode   `yaml:"mode"`
	Hub  string `yaml:"hub,omitempty"`
	// PasswordEnv names an environment variable holding the admin password,
	// so the password itself is never written to the profile file on disk.
	PasswordEnv string `yaml:"password_env,omitempty"`
}

// Address returns the host:port string vpncmd expects as its first argument.
func (p Profile) Address() string {
	return fmt.Sprintf("%s:%d", p.Host, p.Port)
}

// Upsert adds p, or replaces the existing profile with the same Name.
func Upsert(profiles []Profile, p Profile) []Profile {
	for i, existing := range profiles {
		if existing.Name == p.Name {
			profiles[i] = p
			return profiles
		}
	}
	return append(profiles, p)
}

// Remove returns profiles with any entry named name removed.
func Remove(profiles []Profile, name string) []Profile {
	out := make([]Profile, 0, len(profiles))
	for _, p := range profiles {
		if p.Name != name {
			out = append(out, p)
		}
	}
	return out
}

// Store persists profiles to a YAML file on disk.
type Store struct {
	path string
}

// DefaultPath returns the XDG-compliant path for the profiles file,
// e.g. ~/.config/softether-tui/profiles.yaml.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "softether-tui", "profiles.yaml"), nil
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

// DefaultProfiles returns the default set of connection targets (e.g. localhost:443).
func DefaultProfiles() []Profile {
	return []Profile{
		{
			Name: "localhost",
			Host: "localhost",
			Port: 443,
			Mode: ModeServer,
		},
	}
}

// Load reads the profiles file. If the file does not exist, it returns DefaultProfiles().
func (s *Store) Load() ([]Profile, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		defaults := DefaultProfiles()
		_ = s.Save(defaults)
		return defaults, nil
	}
	if err != nil {
		return nil, err
	}
	var profiles []Profile
	if err := yaml.Unmarshal(data, &profiles); err != nil {
		return nil, fmt.Errorf("parse %s: %w", s.path, err)
	}
	if len(profiles) == 0 {
		profiles = DefaultProfiles()
		_ = s.Save(profiles)
	}
	return profiles, nil
}

func (s *Store) Save(profiles []Profile) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(profiles)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}
