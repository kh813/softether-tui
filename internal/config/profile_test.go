package config

import (
	"path/filepath"
	"testing"
)

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.yaml")
	store := NewStore(path)

	want := []Profile{
		{Name: "Tokyo-Prod", Host: "vpn.example.jp", Port: 443, Mode: ModeServer, Hub: "Main"},
		{Name: "Osaka-Bridge", Host: "10.0.2.5", Port: 5555, Mode: ModeBridge},
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d profiles, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("profile %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestStoreLoadMissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	store := NewStore(path)

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d profiles, want 0", len(got))
	}
}

func TestUpsertAddsAndReplaces(t *testing.T) {
	profiles := []Profile{{Name: "a", Host: "1.1.1.1", Port: 443}}

	profiles = Upsert(profiles, Profile{Name: "b", Host: "2.2.2.2", Port: 443})
	if len(profiles) != 2 {
		t.Fatalf("got %d profiles, want 2", len(profiles))
	}

	profiles = Upsert(profiles, Profile{Name: "a", Host: "9.9.9.9", Port: 443})
	if len(profiles) != 2 {
		t.Fatalf("got %d profiles after replace, want 2", len(profiles))
	}
	if profiles[0].Host != "9.9.9.9" {
		t.Errorf("profile 'a' host = %s, want 9.9.9.9", profiles[0].Host)
	}
}

func TestRemove(t *testing.T) {
	profiles := []Profile{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}
	profiles = Remove(profiles, "b")
	if len(profiles) != 2 {
		t.Fatalf("got %d profiles, want 2", len(profiles))
	}
	for _, p := range profiles {
		if p.Name == "b" {
			t.Fatalf("profile 'b' was not removed")
		}
	}
}
