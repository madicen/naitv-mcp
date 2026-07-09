package store

import "errors"

// ErrNameConflict is returned when creating or approving an entry would violate
// the unique-active-name constraint.
var ErrNameConflict = errors.New("store: active entry name already exists")
