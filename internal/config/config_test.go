package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper functions

// compareConfigs compares two Config structs field by field and reports detailed errors.
func compareConfigs(t *testing.T, got, want Config) {
	t.Helper()

	// Compare Prefs
	if got.Prefs.Wrap != want.Prefs.Wrap {
		t.Errorf("Prefs.Wrap = %v, want %v", got.Prefs.Wrap, want.Prefs.Wrap)
	}
	if got.Prefs.Format != want.Prefs.Format {
		t.Errorf("Prefs.Format = %q, want %q", got.Prefs.Format, want.Prefs.Format)
	}
	if !slicesEqual(got.Prefs.Modifiers, want.Prefs.Modifiers) {
		t.Errorf("Prefs.Modifiers = %v, want %v", got.Prefs.Modifiers, want.Prefs.Modifiers)
	}

	// Compare Session
	if got.Session.DeviceID != want.Session.DeviceID {
		t.Errorf("Session.DeviceID = %q, want %q", got.Session.DeviceID, want.Session.DeviceID)
	}
	if got.Session.Pkg != want.Session.Pkg {
		t.Errorf("Session.Pkg = %q, want %q", got.Session.Pkg, want.Session.Pkg)
	}
	if got.Session.Lvl != want.Session.Lvl {
		t.Errorf("Session.Lvl = %q, want %q", got.Session.Lvl, want.Session.Lvl)
	}
	if got.Session.Tag != want.Session.Tag {
		t.Errorf("Session.Tag = %q, want %q", got.Session.Tag, want.Session.Tag)
	}
	if got.Session.Txt != want.Session.Txt {
		t.Errorf("Session.Txt = %q, want %q", got.Session.Txt, want.Session.Txt)
	}
}

// slicesEqual compares two string slices for equality.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// createTestFile creates a file with given content in a temp directory.
func createTestFile(t *testing.T, dir, content string) *os.File {
	t.Helper()

	filePath := filepath.Join(dir, configFileName)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}

	return file
}

// Test functions

func TestDefaultConfig(t *testing.T) {
	got := DefaultConfig()

	// Test Prefs defaults
	t.Run("Prefs", func(t *testing.T) {
		if got.Prefs.Wrap != true {
			t.Errorf("Prefs.Wrap = %v, want true", got.Prefs.Wrap)
		}
		if got.Prefs.Format != "brief" {
			t.Errorf("Prefs.Format = %q, want %q", got.Prefs.Format, "brief")
		}
		wantModifiers := []string{"color"}
		if !slicesEqual(got.Prefs.Modifiers, wantModifiers) {
			t.Errorf("Prefs.Modifiers = %v, want %v", got.Prefs.Modifiers, wantModifiers)
		}
	})

	// Test Session defaults
	t.Run("Session", func(t *testing.T) {
		if got.Session.DeviceID != "" {
			t.Errorf("Session.DeviceID = %q, want empty string", got.Session.DeviceID)
		}
		if got.Session.Pkg != "" {
			t.Errorf("Session.Pkg = %q, want empty string", got.Session.Pkg)
		}
		if got.Session.Lvl != "V" {
			t.Errorf("Session.Lvl = %q, want %q", got.Session.Lvl, "V")
		}
		if got.Session.Tag != "" {
			t.Errorf("Session.Tag = %q, want empty string", got.Session.Tag)
		}
		if got.Session.Txt != "" {
			t.Errorf("Session.Txt = %q, want empty string", got.Session.Txt)
		}
	})

	// Test multiple calls return equal configs
	t.Run("Consistency", func(t *testing.T) {
		config1 := DefaultConfig()
		config2 := DefaultConfig()
		compareConfigs(t, config1, config2)
	})
}

func TestConfig_Save(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "DefaultConfig",
			config: DefaultConfig(),
		},
		{
			name: "CustomPrefs",
			config: Config{
				Prefs: Prefs{
					Wrap:      false,
					Format:    "json",
					Modifiers: []string{"color", "timestamp"},
				},
				Session: Session{
					DeviceID: "",
					Pkg:      "",
					Lvl:      "V",
					Tag:      "",
					Txt:      "",
				},
			},
		},
		{
			name: "CustomSession",
			config: Config{
				Prefs: Prefs{
					Wrap:      true,
					Format:    "brief",
					Modifiers: []string{"color"},
				},
				Session: Session{
					DeviceID: "emulator-5554",
					Pkg:      "com.example.app",
					Lvl:      "D",
					Tag:      "MyTag",
					Txt:      "search text",
				},
			},
		},
		{
			name: "EmptyModifiers",
			config: Config{
				Prefs: Prefs{
					Wrap:      true,
					Format:    "brief",
					Modifiers: []string{},
				},
				Session: Session{
					DeviceID: "",
					Pkg:      "",
					Lvl:      "V",
					Tag:      "",
					Txt:      "",
				},
			},
		},
		{
			name: "SpecialCharactersInSession",
			config: Config{
				Prefs: Prefs{
					Wrap:      true,
					Format:    "brief",
					Modifiers: []string{"color"},
				},
				Session: Session{
					DeviceID: "device-123",
					Pkg:      "com.app.with.dots",
					Lvl:      "E",
					Tag:      "Tag:With:Colons",
					Txt:      "text with \"quotes\" and spaces",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use t.TempDir() to create isolated test directory
			tempDir := t.TempDir()

			// Change to temp directory for this test
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(originalDir)

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Save the config
			path, err := Save(&tt.config)
			if err != nil {
				t.Fatalf("Save() error = %v, want nil", err)
			}

			// Verify path is absolute
			if !filepath.IsAbs(path) {
				t.Errorf("Save() returned relative path %q, want absolute path", path)
			}

			// Verify file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("Config file not created at %q", path)
			}

			// Verify file content is valid JSON
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read config file: %v", err)
			}

			var loadedConfig Config
			if err := json.Unmarshal(content, &loadedConfig); err != nil {
				t.Errorf("Config file contains invalid JSON: %v", err)
			}

			// Verify saved config matches original
			compareConfigs(t, loadedConfig, tt.config)

			// Verify JSON is pretty-printed (has indentation)
			if !bytes.Contains(content, []byte("\n  ")) {
				t.Error("JSON is not indented (expected pretty-print)")
			}

			// Verify file permissions
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("Failed to stat config file: %v", err)
			}
			if info.Mode().Perm() != 0644 {
				t.Errorf("File permissions = %o, want 0644", info.Mode().Perm())
			}
		})
	}
}

func TestConfig_Save_ErrorCases(t *testing.T) {
	t.Run("ReadOnlyDirectory", func(t *testing.T) {
		// Create a read-only directory
		tempDir := t.TempDir()
		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.Mkdir(readOnlyDir, 0555); err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}

		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		if err := os.Chdir(readOnlyDir); err != nil {
			t.Fatalf("Failed to change to read-only directory: %v", err)
		}

		config := DefaultConfig()
		_, err = Save(&config)
		if err == nil {
			t.Error("Save() in read-only directory succeeded, want error")
		}
	})

	t.Run("MultipleSavesOverwrite", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(originalDir)

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("Failed to change to temp directory: %v", err)
		}

		// Save first config
		config1 := DefaultConfig()
		config1.Session.Lvl = "D"
		path1, err := Save(&config1)
		if err != nil {
			t.Fatalf("First Save() error = %v", err)
		}

		// Save second config (should overwrite)
		config2 := DefaultConfig()
		config2.Session.Lvl = "E"
		path2, err := Save(&config2)
		if err != nil {
			t.Fatalf("Second Save() error = %v", err)
		}

		if path1 != path2 {
			t.Errorf("Save paths differ: %q != %q", path1, path2)
		}

		// Verify second config is saved
		file, err := os.Open(path2)
		if err != nil {
			t.Fatalf("Failed to open config file: %v", err)
		}
		defer file.Close()

		loaded, err := Load(file)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if loaded.Session.Lvl != "E" {
			t.Errorf("Loaded config Lvl = %q, want %q", loaded.Session.Lvl, "E")
		}
	})
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name       string
		jsonData   string
		wantConfig Config
	}{
		{
			name: "ValidDefaultConfig",
			jsonData: `{
  "preferences": {
    "soft_wrap": true,
    "log_format": "brief",
    "log_modifiers": ["color"]
  },
  "session": {
    "device_id": "",
    "package_name": "",
    "log_level": "V",
    "log_tag": "",
    "log_text": ""
  }
}`,
			wantConfig: DefaultConfig(),
		},
		{
			name: "ValidCustomConfig",
			jsonData: `{
  "preferences": {
    "soft_wrap": false,
    "log_format": "json",
    "log_modifiers": ["color", "timestamp"]
  },
  "session": {
    "device_id": "emulator-5554",
    "package_name": "com.example.app",
    "log_level": "D",
    "log_tag": "MyTag",
    "log_text": "search"
  }
}`,
			wantConfig: Config{
				Prefs: Prefs{
					Wrap:      false,
					Format:    "json",
					Modifiers: []string{"color", "timestamp"},
				},
				Session: Session{
					DeviceID: "emulator-5554",
					Pkg:      "com.example.app",
					Lvl:      "D",
					Tag:      "MyTag",
					Txt:      "search",
				},
			},
		},
		{
			name:       "CompactJSON",
			jsonData:   `{"preferences":{"soft_wrap":true,"log_format":"brief","log_modifiers":["color"]},"session":{"device_id":"","package_name":"","log_level":"V","log_tag":"","log_text":""}}`,
			wantConfig: DefaultConfig(),
		},
		{
			name: "EmptyModifiersArray",
			jsonData: `{
  "preferences": {
    "soft_wrap": true,
    "log_format": "brief",
    "log_modifiers": []
  },
  "session": {
    "device_id": "",
    "package_name": "",
    "log_level": "V",
    "log_tag": "",
    "log_text": ""
  }
}`,
			wantConfig: Config{
				Prefs: Prefs{
					Wrap:      true,
					Format:    "brief",
					Modifiers: []string{},
				},
				Session: Session{
					DeviceID: "",
					Pkg:      "",
					Lvl:      "V",
					Tag:      "",
					Txt:      "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			file := createTestFile(t, tempDir, tt.jsonData)
			defer file.Close()

			got, err := Load(file)
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}

			compareConfigs(t, got, tt.wantConfig)
		})
	}
}

func TestLoadConfig_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		wantError   bool
		skipExtraOK bool // ExtraFields case doesn't error
	}{
		{
			name:      "EmptyFile",
			jsonData:  "",
			wantError: true,
		},
		{
			name:      "InvalidJSON",
			jsonData:  `{this is not valid json}`,
			wantError: true,
		},
		{
			name:      "IncompleteJSON",
			jsonData:  `{"preferences": {`,
			wantError: true,
		},
		{
			name: "WrongTypes",
			jsonData: `{
  "preferences": {
    "soft_wrap": "not a boolean",
    "log_format": 123,
    "log_modifiers": "not an array"
  },
  "session": {
    "device_id": 456,
    "package_name": true,
    "log_level": [],
    "log_tag": {},
    "log_text": null
  }
}`,
			wantError: true,
		},
		{
			name:      "MissingFields",
			jsonData:  `{"preferences": {}}`,
			wantError: false, // JSON decoder fills with zero values
		},
		{
			name: "NullValues",
			jsonData: `{
  "preferences": null,
  "session": null
}`,
			wantError: false, // JSON decoder fills with zero values
		},
		{
			name: "ExtraFields",
			jsonData: `{
  "preferences": {
    "soft_wrap": true,
    "log_format": "brief",
    "log_modifiers": ["color"],
    "extra_field": "should be ignored"
  },
  "session": {
    "device_id": "",
    "package_name": "",
    "log_level": "V",
    "log_tag": "",
    "log_text": "",
    "another_extra": 123
  }
}`,
			wantError:   false,
			skipExtraOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			file := createTestFile(t, tempDir, tt.jsonData)
			defer file.Close()

			got, err := Load(file)

			// Check error expectation
			if tt.wantError && err == nil {
				t.Errorf("Load() error = nil, want error for %s", tt.name)
			}
			if !tt.wantError && !tt.skipExtraOK && err != nil {
				t.Errorf("Load() error = %v, want nil", err)
			}

			// Even on error, should return default config
			if err != nil {
				want := DefaultConfig()
				compareConfigs(t, got, want)
			}
		})
	}

	t.Run("ClosedFile", func(t *testing.T) {
		tempDir := t.TempDir()
		file := createTestFile(t, tempDir, `{}`)
		file.Close() // Close before reading

		_, err := Load(file)
		if err == nil {
			t.Error("Load() with closed file error = nil, want error")
		}
	})
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "DefaultConfig",
			config: DefaultConfig(),
		},
		{
			name: "FullyPopulatedConfig",
			config: Config{
				Prefs: Prefs{
					Wrap:      false,
					Format:    "threadtime",
					Modifiers: []string{"color", "timestamp", "epoch"},
				},
				Session: Session{
					DeviceID: "device-12345",
					Pkg:      "com.example.myapp",
					Lvl:      "W",
					Tag:      "MainActivity",
					Txt:      "error occurred",
				},
			},
		},
		{
			name: "MinimalConfig",
			config: Config{
				Prefs: Prefs{
					Wrap:      false,
					Format:    "",
					Modifiers: []string{},
				},
				Session: Session{
					DeviceID: "",
					Pkg:      "",
					Lvl:      "",
					Tag:      "",
					Txt:      "",
				},
			},
		},
		{
			name: "UnicodeCharacters",
			config: Config{
				Prefs: Prefs{
					Wrap:      true,
					Format:    "brief",
					Modifiers: []string{"color"},
				},
				Session: Session{
					DeviceID: "デバイス-123",
					Pkg:      "com.例え.app",
					Lvl:      "I",
					Tag:      "тег",
					Txt:      "検索テキスト 🔍",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(originalDir)

			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Save config
			path, err := Save(&tt.config)
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			// Load config
			file, err := os.Open(path)
			if err != nil {
				t.Fatalf("Failed to open saved config: %v", err)
			}
			defer file.Close()

			loaded, err := Load(file)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			// Verify round-trip integrity
			compareConfigs(t, loaded, tt.config)
		})
	}
}

func TestConfig_JSONFieldNames(t *testing.T) {
	// Verify JSON tag mappings are correct
	t.Run("PrefsJSONTags", func(t *testing.T) {
		config := Config{
			Prefs: Prefs{
				Wrap:      false,
				Format:    "test",
				Modifiers: []string{"mod1"},
			},
			Session: Session{},
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		jsonStr := string(data)

		// Verify JSON field names
		expectedFields := []string{
			`"soft_wrap"`,
			`"log_format"`,
			`"log_modifiers"`,
			`"device_id"`,
			`"package_name"`,
			`"log_level"`,
			`"log_tag"`,
			`"log_text"`,
		}

		for _, field := range expectedFields {
			if !strings.Contains(jsonStr, field) {
				t.Errorf("JSON missing expected field %s", field)
			}
		}
	})
}

func TestConfig_SaveFileLocation(t *testing.T) {
	// Verify Save() creates file in current directory
	tempDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	config := DefaultConfig()
	path, err := Save(&config)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file is in current directory
	expectedPath := filepath.Join(tempDir, configFileName)
	if path != expectedPath {
		t.Errorf("Save() path = %q, want %q", path, expectedPath)
	}

	// Verify filename is correct
	if filepath.Base(path) != configFileName {
		t.Errorf("Save() filename = %q, want %q", filepath.Base(path), configFileName)
	}
}

func TestLoadConfig_FilePosition(t *testing.T) {
	// Verify Load can be called multiple times on same file
	tempDir := t.TempDir()
	jsonData := `{
  "preferences": {
    "soft_wrap": true,
    "log_format": "brief",
    "log_modifiers": ["color"]
  },
  "session": {
    "device_id": "test",
    "package_name": "",
    "log_level": "V",
    "log_tag": "",
    "log_text": ""
  }
}`

	file := createTestFile(t, tempDir, jsonData)
	defer file.Close()

	// First load
	config1, err := Load(file)
	if err != nil {
		t.Fatalf("First Load() error = %v", err)
	}

	// Seek back to beginning
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("file.Seek() error = %v", err)
	}

	// Second load
	config2, err := Load(file)
	if err != nil {
		t.Fatalf("Second Load() error = %v", err)
	}

	compareConfigs(t, config1, config2)
}
