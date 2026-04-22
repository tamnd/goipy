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

// Discard removes o from the set if present; no-op otherwise.
func (s *Set) Discard(o Object) {
	h, err := Hash(o)
	if err != nil {
		return
	}
	for i, idx := range s.index[h] {
		eq, err := Eq(s.items[idx], o)
		if err != nil || !eq {
			continue
		}
		// remove from items
		last := len(s.items) - 1
		s.items[idx] = s.items[last]
		s.items = s.items[:last]
		// remove from index bucket
		s.index[h] = append(s.index[h][:i], s.index[h][i+1:]...)
		// update index for the element that was swapped into position idx
		if idx != last {
			mh, _ := Hash(s.items[idx])
			for j, v := range s.index[mh] {
				if v == last {
					s.index[mh][j] = idx
					break
				}
			}
		}
		return
	}
}

// Len returns element count.
func (s *Set) Len() int { return len(s.items) }

// Items exposes the underlying slice (caller must not mutate).
func (s *Set) Items() []Object { return s.items }

// --- Frozenset: same operations, but immutable from Python and hashable. ---

func (s *Frozenset) Add(o Object) error {
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

func (s *Frozenset) Contains(o Object) (bool, error) {
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

func (s *Frozenset) Len() int        { return len(s.items) }
func (s *Frozenset) Items() []Object { return s.items }
