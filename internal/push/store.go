package push

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	webpush "github.com/SherClockHolmes/webpush-go"
)

var ErrInvalid = errors.New("invalid push subscription")

type Keys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

type Subscription struct {
	Endpoint string `json:"endpoint"`
	Keys     Keys   `json:"keys"`
	Sub      string `json:"sub,omitempty"`
}

type diskState struct {
	VAPIDPublic   string         `json:"vapidPublic"`
	VAPIDPrivate  string         `json:"vapidPrivate"`
	Subscriptions []Subscription `json:"subscriptions"`
}

type Store struct {
	mu      sync.Mutex
	path    string
	public  string
	private string
	subs    map[string]Subscription
}

func Open(dir string) (*Store, error) {
	s := &Store{subs: map[string]Subscription{}}
	if dir == "" {
		if err := s.generateKeys(); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s.path = filepath.Join(dir, "state.json")
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) PublicKey() string  { return s.public }
func (s *Store) PrivateKey() string { return s.private }

func (s *Store) Save(sub Subscription) error {
	sub.Endpoint = strings.TrimSpace(sub.Endpoint)
	if sub.Endpoint == "" || sub.Keys.Auth == "" || sub.Keys.P256dh == "" {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[sub.Endpoint] = sub
	return s.writeLocked()
}

func (s *Store) Delete(endpoint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subs, endpoint)
	return s.writeLocked()
}

func (s *Store) List() []Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Subscription, 0, len(s.subs))
	for _, sub := range s.subs {
		out = append(out, sub)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Endpoint < out[j].Endpoint })
	return out
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		if err := s.generateKeys(); err != nil {
			return err
		}
		return s.writeLocked()
	}
	if err != nil {
		return err
	}
	var disk diskState
	if err := json.Unmarshal(b, &disk); err != nil {
		return err
	}
	s.public = disk.VAPIDPublic
	s.private = disk.VAPIDPrivate
	if s.public == "" || s.private == "" {
		if err := s.generateKeys(); err != nil {
			return err
		}
	}
	for _, sub := range disk.Subscriptions {
		if sub.Endpoint == "" {
			continue
		}
		s.subs[sub.Endpoint] = sub
	}
	if s.public != disk.VAPIDPublic || s.private != disk.VAPIDPrivate {
		return s.writeLocked()
	}
	return nil
}

func (s *Store) generateKeys() error {
	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return err
	}
	s.private = private
	s.public = public
	return nil
}

func (s *Store) writeLocked() error {
	if s.path == "" {
		return nil
	}
	disk := diskState{VAPIDPublic: s.public, VAPIDPrivate: s.private}
	for _, sub := range s.subs {
		disk.Subscriptions = append(disk.Subscriptions, sub)
	}
	sort.Slice(disk.Subscriptions, func(i, j int) bool {
		return disk.Subscriptions[i].Endpoint < disk.Subscriptions[j].Endpoint
	})
	b, err := json.MarshalIndent(disk, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
