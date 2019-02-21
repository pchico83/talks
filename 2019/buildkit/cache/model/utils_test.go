package model

import "testing"

func TestGenerateRandomString(t *testing.T) {
	results := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		token := GenerateRandomString(40)
		if _, ok := results[token]; ok {
			t.Fatalf("repeated token was generated: %s", token)
		}

		results[token] = true
	}
}
