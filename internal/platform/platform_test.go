package platform

import "testing"

func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected Platform
	}{
		{
			name:     "Instagram",
			url:      "https://www.instagram.com/p/123/",
			expected: Instagram,
		},
		{
			name:     "Instagram mobile",
			url:      "https://m.instagram.com/reel/123/",
			expected: Instagram,
		},
		{
			name:     "TikTok",
			url:      "https://www.tiktok.com/@usuario/video/123",
			expected: TikTok,
		},
		{
			name:     "TikTok short",
			url:      "https://vm.tiktok.com/ABC123/",
			expected: TikTok,
		},
		{
			name:     "Threads",
			url:      "https://www.threads.net/@usuario/post/123",
			expected: Threads,
		},
		{
			name:     "Threads .com",
			url:      "https://" + "www.threads.com/@usuario/post/123",
			expected: Threads,
		},
		{
			name:     "X",
			url:      "https://x.com/usuario/status/123",
			expected: Twitter,
		},
		{
			name:     "Twitter",
			url:      "https://twitter.com/usuario/status/123",
			expected: Twitter,
		},
		{
			name:     "Reddit",
			url:      "https://www.reddit.com/r/teste/comments/123/",
			expected: Reddit,
		},
		{
			name:     "Reddit short",
			url:      "https://redd.it/123",
			expected: Reddit,
		},
		{
			name:     "YouTube normal não suportado",
			url:      "https://www.youtube.com/watch?v=ABC123",
			expected: Unknown,
		},
		{
			name:     "YouTube Shorts",
			url:      "https://www.youtube.com/shorts/123",
			expected: YouTube,
		},
		{
			name:     "YouTube youtu.be não suportado",
			url:      "https://youtu.be/ABC123",
			expected: Unknown,
		},
		{
			name:     "Erome",
			url:      "https://www.erome.com/a/123",
			expected: Erome,
		},
		{
			name:     "Erome regional",
			url:      "https://pt.erome.com/a/123",
			expected: Erome,
		},
		{
			name:     "Threads share",
			url:      "https://www.threads.com/share/ABC123/",
			expected: Threads,
		},
		{
			name:     "Reddit share",
			url:      "https://www.reddit.com/r/teste/s/ABC123",
			expected: Reddit,
		},
		{
			name:     "Plataforma desconhecida",
			url:      "https://example.com/video",
			expected: Unknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Detect(test.url)

			if result != test.expected {
				t.Fatalf(
					"Detect(%q) = %q; esperado %q",
					test.url,
					result,
					test.expected,
				)
			}
		})
	}
}
