package scraper

import (
	"testing"
)

func TestDetectLicenseType(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "MIT License",
			content:  "MIT License\n\nPermission is hereby granted, free of charge...",
			expected: "MIT",
		},
		{
			name:     "Apache 2.0 License",
			content:  "Apache License Version 2.0, January 2004",
			expected: "Apache-2.0",
		},
		{
			name:     "GPL-2.0",
			content:  "GNU GENERAL PUBLIC LICENSE Version 2, June 1991",
			expected: "GPL-2.0",
		},
		{
			name:     "GPL-3.0",
			content:  "GNU GENERAL PUBLIC LICENSE Version 3, 29 June 2007",
			expected: "GPL-3.0",
		},
		{
			name:     "LGPL-2.1",
			content:  "GNU LESSER GENERAL PUBLIC LICENSE Version 2.1, February 1999",
			expected: "LGPL-2.1",
		},
		{
			name:     "LGPL-3.0",
			content:  "GNU LESSER GENERAL PUBLIC LICENSE Version 3, 29 June 2007",
			expected: "LGPL-3.0",
		},
		{
			name: "ISC License",
			content: `ISC License (ISC)
			Permission to use, copy, modify, and/or distribute this software for any purpose`,
			expected: "ISC",
		},
		{
			name: "BSD-3-Clause",
			content: `Redistribution and use in source and binary forms, with or without modification,
			are permitted provided that the following three clauses are met`,
			expected: "BSD-3-Clause",
		},
		{
			name: "BSD-2-Clause",
			content: `Redistribution and use in source and binary forms, with or without modification,
			are permitted provided that the following two clauses are met`,
			expected: "BSD-2-Clause",
		},
		{
			name:     "Unlicense",
			content:  "This is free and unencumbered software released into the public domain.",
			expected: "Unlicense",
		},
		{
			name:     "MPL-2.0",
			content:  `Mozilla Public License Version 2.0`,
			expected: "MPL-2.0",
		},
		{
			name:     "AGPL-3.0",
			content:  `GNU Affero General Public License Version 3`,
			expected: "AGPL-3.0",
		},
		{
			name:     "EPL-2.0",
			content:  `Eclipse Public License Version 2.0`,
			expected: "EPL-2.0",
		},
		{
			name:     "CC0-1.0",
			content:  `Creative Commons Zero`,
			expected: "CC0-1.0",
		},
		{
			name: "GPL-2.0 or later variant",
			content: `GNU GENERAL PUBLIC LICENSE
			Version 2, June 1991
			or (at your option) any later version`,
			expected: "GPL-2.0+",
		},
		{
			name: "GPL-3.0 or later variant",
			content: `GNU GENERAL PUBLIC LICENSE
			Version 3, 29 June 2007
			or any later version of the License`,
			expected: "GPL-3.0+",
		},
		{
			name:     "Unknown License",
			content:  "Some custom license text that doesn't match any patterns",
			expected: "Unknown",
		},
		{
			name:     "Empty content",
			content:  "",
			expected: "Unknown",
		},
		{
			name: "Case insensitive MIT",
			content: `mit license
			permission is hereby granted`,
			expected: "MIT",
		},
		{
			name:     "Zlib License",
			content:  `zlib License`,
			expected: "Zlib",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectLicenseType(tt.content)
			if result != tt.expected {
				t.Errorf("DetectLicenseType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLicenseFileNames(t *testing.T) {
	names := LicenseFileNames()

	expectedNames := map[string]bool{
		"LICENSE":     true,
		"LICENSE.md":  true,
		"LICENSE.txt": true,
		"COPYING":     true,
		"COPYING.md":  true,
		"COPYING.txt": true,
		"LICENSE.rst": true,
	}

	if len(names) != len(expectedNames) {
		t.Errorf("LicenseFileNames() returned %d names, expected %d", len(names), len(expectedNames))
	}

	for _, name := range names {
		if !expectedNames[name] {
			t.Errorf("LicenseFileNames() returned unexpected name: %q", name)
		}
	}

	// Check that LICENSE is first (tried first)
	if len(names) > 0 && names[0] != "LICENSE" {
		t.Errorf("LicenseFileNames() first element is %q, expected LICENSE", names[0])
	}
}

func BenchmarkDetectLicenseType(b *testing.B) {
	content := `MIT License

Copyright (c) 2024 Example Author

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectLicenseType(content)
	}
}
