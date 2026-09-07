package cuenta_corriente

import "testing"

func TestParseDueDate(t *testing.T) {
	valid := []string{"2026-09-01", "2001-02-03"}
	for _, s := range valid {
		d, err := parseDueDate(s)
		if err != nil {
			t.Fatalf("parseDueDate(%q): err=%v", s, err)
		}
		if !d.Valid {
			t.Fatalf("parseDueDate(%q): Valid=false", s)
		}
		if got := d.Time.Format("2006-01-02"); got != s {
			t.Fatalf("parseDueDate(%q): got %q", s, got)
		}
	}

	invalid := []string{"", "not-a-date", "2026/09/01", "2026-09-0"}
	for _, s := range invalid {
		if _, err := parseDueDate(s); err == nil {
			t.Fatalf("parseDueDate(%q): expected error, got nil", s)
		}
	}
}

func TestParseUUID(t *testing.T) {
	s := "4f7ac6c1-e6d3-4d1a-8c1a-0d9a7e0a1b2c"
	u, err := parseUUID(s)
	if err != nil {
		t.Fatalf("parseUUID: err=%v", err)
	}
	if !u.Valid {
		t.Fatal("parseUUID: Valid=false")
	}
	if got := u.String(); got != s {
		t.Fatalf("parseUUID: got %q want %q", got, s)
	}

	if _, err := parseUUID("no-es-uuid"); err == nil {
		t.Fatal("parseUUID(invalid): expected error, got nil")
	}
}

func TestParseLiquidacionID(t *testing.T) {
	if _, err := parseLiquidacionID(nil); err != nil {
		t.Fatalf("parseLiquidacionID(nil): err=%v", err)
	}
	s := "4f7ac6c1-e6d3-4d1a-8c1a-0d9a7e0a1b2c"
	u, err := parseLiquidacionID(&s)
	if err != nil {
		t.Fatalf("parseLiquidacionID: err=%v", err)
	}
	if !u.Valid || u.String() != s {
		t.Fatalf("parseLiquidacionID: got valid=%v value=%q", u.Valid, u.String())
	}
	bad := "mal"
	if _, err := parseLiquidacionID(&bad); err == nil {
		t.Fatal("parseLiquidacionID(invalid): expected error, got nil")
	}
}