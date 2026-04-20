package object

// Add inserts o into the set if not already present.
func (s *Set) Add(o Object) error {
	h, err := Hash(o)
	if err != nil {
		return err
	}
	for _, idx := range s.index[h] {
		eq, err := Eq(s.items[idx], o)
		if err != nil {
			return err
		}
		if eq {
			return nil
		}
	}
	s.index[h] = append(s.index[h], len(s.items))
	s.items = append(s.items, o)
	return nil
}

// Contains tests membership.
func (s *Set) Contains(o Object) (bool, error) {
	h, err := Hash(o)
	if err != nil {
		return false, err
	}
	for _, idx := range s.index[h] {
		eq, err := Eq(s.items[idx], o)
		if err != nil {
			return false, err
		}
		if eq {
			return true, nil
		}
	}
	return false, nil
}

// Len returns element count.
func (s *Set) Len() int { return len(s.items) }

// Items exposes the underlying slice (caller must not mutate).
func (s *Set) Items() []Object { return s.items }
