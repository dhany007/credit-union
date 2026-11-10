package loan

import (
	"encoding/json"
	"testing"
)

func TestMinimumLoanPolicyRate(t *testing.T) {
	cases := []struct {
		name, body string
		valid      bool
	}{
		{"minimum", `{"base_rate_bps":300,"bands":[],"reason":"Minimum tariff"}`, true},
		{"below minimum", `{"base_rate_bps":299,"bands":[],"reason":"Minimum tariff"}`, false},
		{"missing base", `{"bands":[],"reason":"Minimum tariff"}`, false},
		{"null base", `{"base_rate_bps":null,"bands":[],"reason":"Minimum tariff"}`, false},
		{"zero base", `{"base_rate_bps":0,"bands":[],"reason":"Minimum tariff"}`, false},
		{"minimum band", `{"base_rate_bps":300,"bands":[{"min_amount":0,"max_amount":null,"rate_bps":300}],
"reason":"Minimum tariff"}`, true},
		{"below minimum band", `{"base_rate_bps":300,"bands":[{"min_amount":0,"max_amount":null,"rate_bps":299}],
"reason":"Minimum tariff"}`, false},
		{
			"missing band rate",
			`{"base_rate_bps":300,"bands":[{"min_amount":0,"max_amount":null}],"reason":"Minimum tariff"}`,
			false,
		},
		{"null band rate", `{"base_rate_bps":300,"bands":[{"min_amount":0,"max_amount":null,"rate_bps":null}],
"reason":"Minimum tariff"}`, false},
		{"missing bands", `{"base_rate_bps":300,"reason":"Minimum tariff"}`, false},
		{"null bands", `{"base_rate_bps":300,"bands":null,"reason":"Minimum tariff"}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var p Policy
			if err := json.Unmarshal([]byte(c.body), &p); err != nil {
				t.Fatal(err)
			}
			if got := p.validate() == nil; got != c.valid {
				t.Fatalf("policy valid=%v want %v", got, c.valid)
			}
		})
	}
}
