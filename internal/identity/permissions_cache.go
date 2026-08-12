package identity

import (
	"sync"
	"time"
)

// membershipEntry son los roles+scopes resueltos de una membresía, listos para
// derivar permisos sin volver a consultar la base.
type membershipEntry struct {
	roles  []string
	scopes []string
	expiry time.Time
}

// membershipCache es un caché TTL en memoria de roles/scopes por membresía
// (clave userID:membershipID). Reduce las consultas de autorización por
// request; el TTL corto limita la ventana de staleness ante cambios de roles.
type membershipCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	now   func() time.Time
	items map[string]membershipEntry
}

func newMembershipCache(ttl time.Duration) *membershipCache {
	return &membershipCache{
		ttl:   ttl,
		now:   time.Now,
		items: make(map[string]membershipEntry),
	}
}

func membershipCacheKey(userID, membershipID string) string {
	return userID + ":" + membershipID
}

// get devuelve roles/scopes si hay una entrada vigente; con TTL <= 0 siempre miss.
func (c *membershipCache) get(key string) (roles, scopes []string, ok bool) {
	if c.ttl <= 0 {
		return nil, nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, found := c.items[key]
	if !found || !c.now().Before(e.expiry) {
		return nil, nil, false
	}
	return e.roles, e.scopes, true
}

func (c *membershipCache) set(key string, roles, scopes []string) {
	if c.ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = membershipEntry{
		roles:  roles,
		scopes: scopes,
		expiry: c.now().Add(c.ttl),
	}
}
