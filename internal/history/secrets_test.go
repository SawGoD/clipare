package history

import "testing"

func TestSecretsDetected(t *testing.T) {
	cases := []string{
		"ghp_0123456789abcdefghijklmnopqrstuvwxyzAB",
		"github_pat_0123456789abcdefghijklmnopqrstuvwxyz",
		"sk-0123456789abcdefghijklmnopqrstuvwxyz",
		"Bearer 0123456789abcdefghijklmnopqrstuvwxyzAB",
		"bearer 0123456789abcdefghijklmnopqrstuvwxyzAB",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
		"AKIAIOSFODNN7EXAMPLE",
		"-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7\n-----END PRIVATE KEY-----",
		"0123456789abcdef0123456789abcdef01234567",
	}
	for _, text := range cases {
		if !LooksSensitive(text) {
			t.Fatalf("secret not detected: %q", text)
		}
	}
}

func TestSecretsNotDetected(t *testing.T) {
	cases := []string{
		"Сегодня хорошая погода, пойдём гулять в парк после работы",
		"https://github.com/SawGoD/clipare",
		"https://example.com/very/long/path/with/many/segments/0123456789",
		"docker compose ps -a --format json | head -20",
		"This is a long regular prose sentence about many things that just happens to be quite long overall.",
		"0123456789",
		" Meeting notes from Monday: budgets, roadmap, planning",
	}
	for _, text := range cases {
		if LooksSensitive(text) {
			t.Fatalf("ordinary text flagged as secret: %q", text)
		}
	}
}

func TestSecretsMultiLineProseSafe(t *testing.T) {
	prose := "Первая строка обычного текста\nВторая строка тоже обычная\nИ третья строка не секрет"
	if LooksSensitive(prose) {
		t.Fatal("multi-line prose flagged")
	}
	if !LooksSensitive("безопасно\nAKIAIOSFODNN7EXAMPLE\nконец") {
		t.Fatal("secret inside multi-line text missed")
	}
}
