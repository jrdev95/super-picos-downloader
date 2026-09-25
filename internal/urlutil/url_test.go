package urlutil

import "testing"

func TestExtractFirst(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		found    bool
	}{
		{
			name:     "URL pura",
			input:    "https://www.tiktok.com/@usuario/video/123",
			expected: "https://www.tiktok.com/@usuario/video/123",
			found:    true,
		},
		{
			name: "URL com parâmetros",
			input: "https://www.tiktok.com/@usuario/photo/123" +
				"?is_from_webapp=1&sender_device=pc",
			expected: "https://www.tiktok.com/@usuario/photo/123" +
				"?is_from_webapp=1&sender_device=pc",
			found: true,
		},
		{
			name:     "URL dentro de texto",
			input:    "Veja isso https://www.instagram.com/reel/123 agora",
			expected: "https://www.instagram.com/reel/123",
			found:    true,
		},
		{
			name: "Markdown",
			input: "[https://www.tiktok.com/@usuario/photo/123]" +
				"(https://www.tiktok.com/@usuario/photo/123)",
			expected: "https://www.tiktok.com/@usuario/photo/123",
			found:    true,
		},
		{
			name: "Markdown aninhado",
			input: "[[https://www.tiktok.com/@usuario/photo/123]" +
				"(https://www.tiktok.com/@usuario/photo/123)]" +
				"(https://www.tiktok.com/@usuario/photo/123)",
			expected: "https://www.tiktok.com/@usuario/photo/123",
			found:    true,
		},
		{
			name:  "Sem URL",
			input: "nenhum link aqui",
			found: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, found := ExtractFirst(test.input)

			if found != test.found {
				t.Fatalf(
					"found = %t; esperado %t",
					found,
					test.found,
				)
			}

			if result != test.expected {
				t.Fatalf(
					"resultado = %q; esperado %q",
					result,
					test.expected,
				)
			}
		})
	}
}
