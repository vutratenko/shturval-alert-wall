package push

import (
	"testing"
)

func TestStore_upsertAndDelete(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sub := Subscription{Endpoint: "https://push.example/a", Keys: Keys{P256dh: "p", Auth: "a"}, Sub: "user"}
	if err := s.Save(sub); err != nil {
		t.Fatal(err)
	}
	sub.Sub = "user-2"
	if err := s.Save(sub); err != nil {
		t.Fatal(err)
	}
	got := s.List()
	if len(got) != 1 || got[0].Sub != "user-2" {
		t.Fatalf("%+v", got)
	}
	if err := s.Delete(sub.Endpoint); err != nil {
		t.Fatal(err)
	}
	if len(s.List()) != 0 {
		t.Fatal("subscription remained")
	}
	if err := s.Save(Subscription{Endpoint: "x"}); err != ErrInvalid {
		t.Fatalf("err %v", err)
	}
}

func TestStore_vapidSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	first, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Save(Subscription{Endpoint: "https://push.example/a", Keys: Keys{P256dh: "p", Auth: "a"}}); err != nil {
		t.Fatal(err)
	}
	second, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if second.PublicKey() == "" || second.PublicKey() != first.PublicKey() || second.PrivateKey() != first.PrivateKey() {
		t.Fatal("vapid keys changed")
	}
	if len(second.List()) != 1 {
		t.Fatal("subscription lost")
	}
}

func TestStore_memoryDoesNotShareKeys(t *testing.T) {
	a, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	if a.PublicKey() == "" || a.PublicKey() == b.PublicKey() {
		t.Fatal("expected a fresh in-memory key")
	}
}
