package testutils

import (
	"fmt"
	"sync/atomic"
)

var idCounter int64

// NewID returns a deterministic, unique fake ID string for use as a primary
// key in tests, avoiding a dependency on the database's gen_random_uuid()
// default (which sqlmock never actually executes).
func NewID(prefix string) string {
	n := atomic.AddInt64(&idCounter, 1)
	return fmt.Sprintf("test-%s-%d", prefix, n)
}
