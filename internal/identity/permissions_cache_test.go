package identity

import (
	"testing"
	"time"
)

func TestMembershipCacheSetGet(t *testing.T) {
	c := newMembershipCache(time.Minute)
	key := membershipCacheKey("user-1", "mem-1")

	roles, scopes, ok := c.get(key)
	if ok {
		t.Fatalf("get sobre caché vacío debería ser miss, fue hit (%v/%v)", roles, scopes)
	}

	c.set(key, []string{"tesorero"}, []string{"consorcio:1"})
	roles, scopes, ok = c.get(key)
	if !ok {
		t.Fatal("get tras set debería ser hit")
	}
	if len(roles) != 1 || roles[0] != "tesorero" {
		t.Fatalf("roles = %v, se esperaba [tesorero]", roles)
	}
	if len(scopes) != 1 || scopes[0] != "consorcio:1" {
		t.Fatalf("scopes = %v, se esperaba [consorcio:1]", scopes)
	}
}

func TestMembershipCacheExpira(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	c := newMembershipCache(time.Minute)
	c.now = func() time.Time { return now }

	key := membershipCacheKey("user-1", "mem-1")
	c.set(key, []string{"auditor"}, nil)

	now = now.Add(time.Second)
	if _, _, ok := c.get(key); !ok {
		t.Fatal("antes del TTL debería ser hit")
	}

	now = now.Add(time.Minute)
	if _, _, ok := c.get(key); ok {
		t.Fatal("después del TTL debería ser miss")
	}
}

func TestMembershipCacheTTLCeroDesactiva(t *testing.T) {
	c := newMembershipCache(0)
	key := membershipCacheKey("user-1", "mem-1")
	c.set(key, []string{"tesorero"}, nil)
	if _, _, ok := c.get(key); ok {
		t.Fatal("con TTL=0 el caché debe estar desactivado")
	}
}

func TestMembershipCacheClavesAisladas(t *testing.T) {
	c := newMembershipCache(time.Minute)
	c.set(membershipCacheKey("user-1", "mem-1"), []string{"tesorero"}, nil)
	roles, _, ok := c.get(membershipCacheKey("user-1", "mem-2"))
	if ok {
		t.Fatalf("otra membresía del mismo usuario no debe ser hit (%v)", roles)
	}
}
