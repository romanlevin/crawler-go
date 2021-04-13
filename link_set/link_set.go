package link_set

import "sync"

// LinkSet is a concurrent hashset of strings, capable of adding new strings and of checking for the presence of strings
type LinkSet struct {
	store map[string]struct{}
	lock  sync.RWMutex
}

func New() *LinkSet {
	return &LinkSet{
		store: make(map[string]struct{}),
		lock:  sync.RWMutex{},
	}
}

func (s *LinkSet) Has(link string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok := s.store[link]
	return ok
}

func (s *LinkSet) Add(link string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.store[link] = struct{}{}
}
