package dashboard

import "testing"

func TestFirstCurrency(t *testing.T) {
	t.Run("returns one existing key", func(t *testing.T) {
		currencies := map[string]struct{}{"NOK": {}, "USD": {}}
		got := firstCurrency(currencies)
		if got == "" {
			t.Fatalf("expected non-empty currency code")
		}
		if _, ok := currencies[got]; !ok {
			t.Fatalf("returned key %q is not in input map", got)
		}
	})

	t.Run("empty map returns empty string", func(t *testing.T) {
		got := firstCurrency(map[string]struct{}{})
		if got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})
}
