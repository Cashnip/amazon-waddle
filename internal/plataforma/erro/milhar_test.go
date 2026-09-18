package erro

import "testing"

func TestMilhar(t *testing.T) {
	for n, quero := range map[int64]string{0: "0", 999: "999", 1000: "1.000", 4000: "4.000", 100000000: "100.000.000", -1234: "-1.234"} {
		if got := Milhar(n); got != quero {
			t.Errorf("Milhar(%d) = %q, quero %q", n, got, quero)
		}
	}
}
