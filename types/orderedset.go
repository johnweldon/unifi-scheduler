package types

import (
	"sync"
)

type OrderedStringSet struct {
	sync.RWMutex
	collection []string
	guard      map[string]interface{}
}

func (s *OrderedStringSet) Add(v ...string) bool {
	if len(v) == 0 {
		return false
	}

	s.Lock()
	defer s.Unlock()

	if s.guard == nil {
		s.guard = map[string]interface{}{}
	}

	for _, val := range v {
		if _, ok := s.guard[val]; ok {
			return false
		}

		s.guard[val] = nil
	}

	s.collection = append(s.collection, v...)

	return true
}

func (s *OrderedStringSet) Remove(v ...string) bool {
	if len(v) == 0 {
		return false
	}

	s.Lock()
	defer s.Unlock()

	if s.guard == nil {
		s.guard = map[string]interface{}{}
	}

	for _, val := range v {
		if _, ok := s.guard[val]; !ok {
			return false
		}

		if _, ok := indexOf(val, s.collection); !ok {
			return false
		}
	}

	for _, val := range v {
		delete(s.guard, val)

		ix, ok := indexOf(val, s.collection)
		if !ok { // should never happen, but here for safety.
			return false
		}

		s.collection = append(s.collection[:ix], s.collection[ix+1:]...)
	}

	return true
}

func (s *OrderedStringSet) Values() []string {
	s.RLock()
	defer s.RUnlock()

	res := make([]string, len(s.collection))
	for ix, val := range s.collection {
		res[ix] = val
	}

	return res
}

func indexOf(v string, coll []string) (int, bool) {
	for ix, item := range coll {
		if item == v {
			return ix, true
		}
	}

	return -1, false
}
