package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateScriptName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{"valid alphanumeric", "validScript123", false, ""},
		{"valid with underscore", "my_script", false, ""},
		{"valid with hyphen", "my-script", false, ""},
		{"empty name", "", true, "must be between"},
		{"too long", strings.Repeat("a", 65), true, "must be between"},
		{"with spaces", "my script", true, "only alphanumeric"},
		{"with dots", "my.script", true, "only alphanumeric"},
		{"with path traversal", "../evil", true, "invalid path characters"},
		{"with forward slash", "my/script", true, "invalid path characters"},
		{"with backslash", "my\\script", true, "invalid path characters"},
		{"with double dots", "my..script", true, "invalid path characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScriptName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateScriptName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateScriptName(%q) error = %v, want error containing %q", tt.input, err, tt.errMsg)
			}
		})
	}
}

func TestSecureFilePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		scriptsDir string
		filename   string
		wantErr    bool
		errMsg     string
	}{
		{"valid filename", tempDir, "script.ts", false, ""},
		{"path traversal attempt", tempDir, "../../../etc/passwd", true, "path traversal"},
		{"absolute path", tempDir, "/etc/passwd", true, "path traversal"},
		{"with subdirectory", tempDir, "subdir/script.ts", true, "path traversal"},
		{"double dots in middle", tempDir, "sc..ript.ts", false, ""}, // This is actually valid as a filename
		{"backslash", tempDir, "script\\bad.ts", true, "path traversal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := secureFilePath(tt.scriptsDir, tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("secureFilePath(%q, %q) error = %v, wantErr %v", tt.scriptsDir, tt.filename, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("secureFilePath(%q, %q) error = %v, want error containing %q", tt.scriptsDir, tt.filename, err, tt.errMsg)
			}
		})
	}
}

func TestParseScriptFile(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Test cases
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, script *Script)
	}{
		{
			name: "valid script",
			content: `/***
{
  "description": "Test script",
  "category": "test",
  "tags": ["test", "example"],
  "author": "Test Author",
  "version": "1.0.0"
}
***/

function test() { return "hello"; }`,
			wantErr: false,
			check: func(t *testing.T, script *Script) {
				if script.Metadata.Description != "Test script" {
					t.Errorf("got description %q, want %q", script.Metadata.Description, "Test script")
				}
				if script.Metadata.Category != "test" {
					t.Errorf("got category %q, want %q", script.Metadata.Category, "test")
				}
				if len(script.Metadata.Tags) != 2 {
					t.Errorf("got %d tags, want 2", len(script.Metadata.Tags))
				}
			},
		},
		{
			name:    "missing front matter",
			content: `function test() { return "hello"; }`,
			wantErr: true,
		},
		{
			name: "invalid JSON in front matter",
			content: `/***
{ invalid json }
***/

function test() { return "hello"; }`,
			wantErr: true,
		},
		{
			name: "missing description",
			content: `/***
{
  "category": "test"
}
***/

function test() { return "hello"; }`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filename := "test_script.ts"
			filePath := filepath.Join(tempDir, filename)
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Parse the file
			script, err := parseScriptFile(filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseScriptFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, script)
			}
		})
	}
}

func TestSanitizeJavaScript(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no changes needed", "function test() { return 42; }", "function test() { return 42; }"},
		{"remove script tags", "code<script>evil</script>more", "codemore"},
		{"escape backticks", "const str = `hello`;", "const str = \\`hello\\`;"},
		{"remove HTML comments", "code<!--comment-->more", "codemore"},
		{"multiple sanitizations", "<script>`test`</script><!--bad-->", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeJavaScript(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeJavaScript() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEscapeForJavaScript(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no escaping needed", "simple text", "simple text"},
		{"escape quotes", `"quoted"`, `\"quoted\"`},
		{"escape single quotes", `'quoted'`, `\'quoted\'`},
		{"escape backslashes", `path\to\file`, `path\\to\\file`},
		{"escape newlines", "line1\nline2", `line1\nline2`},
		{"escape tabs", "col1\tcol2", `col1\tcol2`},
		{"complex escaping", `"test"\n'value'`, `\"test\"\\n\'value\'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeForJavaScript(tt.input)
			if got != tt.want {
				t.Errorf("escapeForJavaScript() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSaveAndRemoveScript(t *testing.T) {
	// Set up test scripts directory
	tempDir := t.TempDir()
	oldGetScriptsDir := getScriptsDirectory
	getScriptsDirectory = func() (string, error) {
		return tempDir, nil
	}
	defer func() { getScriptsDirectory = oldGetScriptsDir }()

	// Test save script
	metadata := ScriptMetadata{
		Description: "Test save script",
		Category:    "test",
		Tags:        []string{"test"},
		Author:      "Test",
		Version:     "1.0.0",
	}

	code := "function test() { return 'saved'; }"
	scriptName := "test_save_script"

	// Save the script
	err := saveScript(scriptName, code, metadata)
	if err != nil {
		t.Fatalf("saveScript() error = %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tempDir, scriptName+".ts")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("script file was not created at %s", expectedPath)
	}

	// Test remove script
	err = removeScript(scriptName)
	if err != nil {
		t.Fatalf("removeScript() error = %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("script file was not removed from %s", expectedPath)
	}

	// Test removing non-existent script
	err = removeScript("non_existent")
	if err == nil {
		t.Errorf("removeScript() should error for non-existent script")
	}
}

func TestGenerateLibraryCode(t *testing.T) {
	scripts := []Script{
		{
			Name: "testFunc",
			Metadata: ScriptMetadata{
				Description: "Test function",
				Category:    "test",
				Tags:        []string{"test", "example"},
				Examples:    []string{"testFunc()", "testFunc(42)"},
				Parameters:  map[string]string{"value": "input value"},
				ReturnType:  "string",
				Author:      "Test Author",
				Version:     "1.0.0",
			},
			Code: "function testFunc(value) { return 'test: ' + value; }",
		},
	}

	code := generateLibraryCode(scripts)

	// Check that the generated code contains expected elements
	checks := []string{
		"window.brummerLibrary = {",
		"testFunc:",
		"Test function",
		"window.lib = window.brummerLibrary",
		"__meta:",
		"list: function()",
		"help: function(",
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("generated code missing expected element: %q", check)
		}
	}

	// Verify it's valid JavaScript by checking basic syntax
	if strings.Count(code, "{") != strings.Count(code, "}") {
		t.Errorf("generated code has unbalanced braces")
	}
}

func TestLoadAllScripts(t *testing.T) {
	// Set up test scripts directory
	tempDir := t.TempDir()
	oldGetScriptsDir := getScriptsDirectory
	getScriptsDirectory = func() (string, error) {
		return tempDir, nil
	}
	defer func() { getScriptsDirectory = oldGetScriptsDir }()

	// Create test scripts
	script1 := `/***
{
  "description": "Script 1"
}
***/
function script1() { return 1; }`

	script2 := `/***
{
  "description": "Script 2",
  "category": "util"
}
***/
function script2() { return 2; }`

	// Write scripts
	os.WriteFile(filepath.Join(tempDir, "script1.ts"), []byte(script1), 0644)
	os.WriteFile(filepath.Join(tempDir, "script2.ts"), []byte(script2), 0644)

	// Also create a non-.ts file that should be ignored
	os.WriteFile(filepath.Join(tempDir, "notscript.js"), []byte("// not a ts file"), 0644)

	// Load all scripts
	scripts, err := loadAllScripts()
	if err != nil {
		t.Fatalf("loadAllScripts() error = %v", err)
	}

	// Verify correct number of scripts loaded
	if len(scripts) != 2 {
		t.Errorf("got %d scripts, want 2", len(scripts))
	}

	// Verify script details
	scriptNames := make(map[string]bool)
	for _, s := range scripts {
		scriptNames[s.Name] = true
	}

	if !scriptNames["script1"] || !scriptNames["script2"] {
		t.Errorf("expected scripts not found, got: %v", scriptNames)
	}
}
