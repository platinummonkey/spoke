package protobuf

import (
	"bufio"
	"strings"
	"testing"
)

func TestNewScanner(t *testing.T) {
	input := "syntax = \"proto3\";"
	scanner := NewScanner(strings.NewReader(input))

	if scanner == nil {
		t.Fatal("Expected scanner to be created")
	}

	if scanner.line != 1 {
		t.Errorf("Expected line 1, got %d", scanner.line)
	}

	if scanner.column != 2 {
		t.Errorf("Expected column 2 after initialization, got %d", scanner.column)
	}
}

func TestScannerBasicTokens(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			text      string
		}
	}{
		{
			name:  "identifier",
			input: "syntax",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenIdentifier, "syntax"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "identifier with underscore",
			input: "my_field",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenIdentifier, "my_field"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "identifier with dot",
			input: "google.protobuf.Timestamp",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenIdentifier, "google.protobuf.Timestamp"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "number",
			input: "123",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenNumber, "123"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "decimal number",
			input: "3.14",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenNumber, "3.14"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "scientific notation",
			input: "1.23e10",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenNumber, "1.23e10"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "scientific notation with negative exponent",
			input: "1.5E-3",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenNumber, "1.5E-3"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "punctuation semicolon",
			input: ";",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenPunctuation, ";"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "punctuation braces",
			input: "{}",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenPunctuation, "{"},
				{TokenPunctuation, "}"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "punctuation brackets",
			input: "[]",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenPunctuation, "["},
				{TokenPunctuation, "]"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "punctuation parentheses",
			input: "()",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenPunctuation, "("},
				{TokenPunctuation, ")"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "punctuation angle brackets",
			input: "<>",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenPunctuation, "<"},
				{TokenPunctuation, ">"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "punctuation colon",
			input: ":",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenPunctuation, ":"},
				{TokenEOF, ""},
			},
		},
		{
			name:  "punctuation comma",
			input: ",",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenPunctuation, ","},
				{TokenEOF, ""},
			},
		},
		{
			name:  "punctuation equals",
			input: "=",
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenPunctuation, "="},
				{TokenEOF, ""},
			},
		},
		{
			name:  "double quoted string",
			input: `"hello"`,
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenString, `"hello"`},
				{TokenEOF, ""},
			},
		},
		{
			name:  "single quoted string",
			input: `'world'`,
			expected: []struct {
				tokenType TokenType
				text      string
			}{
				{TokenString, `"world"`},
				{TokenEOF, ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tc.input))

			for i, expected := range tc.expected {
				tok, err := scanner.Scan()
				if err != nil && expected.tokenType != TokenError {
					t.Fatalf("Token %d: unexpected error: %v", i, err)
				}

				if tok.Type != expected.tokenType {
					t.Errorf("Token %d: expected type %q, got %q", i, expected.tokenType, tok.Type)
				}

				if tok.Text != expected.text {
					t.Errorf("Token %d: expected text %q, got %q", i, expected.text, tok.Text)
				}
			}
		})
	}
}

func TestScanStringEscapeSequences(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedText  string
		expectError   bool
	}{
		{
			name:         "newline escape",
			input:        `"hello\nworld"`,
			expectedText: "\"hello\nworld\"",
			expectError:  false,
		},
		{
			name:         "tab escape",
			input:        `"hello\tworld"`,
			expectedText: "\"hello\tworld\"",
			expectError:  false,
		},
		{
			name:         "carriage return escape",
			input:        `"hello\rworld"`,
			expectedText: "\"hello\rworld\"",
			expectError:  false,
		},
		{
			name:         "backslash escape",
			input:        `"hello\\world"`,
			expectedText: "\"hello\\world\"",
			expectError:  false,
		},
		{
			name:         "double quote escape",
			input:        `"hello\"world"`,
			expectedText: "\"hello\"world\"",
			expectError:  false,
		},
		{
			name:         "single quote escape",
			input:        `"hello\'world"`,
			expectedText: "\"hello'world\"",
			expectError:  false,
		},
		{
			name:         "unicode escape",
			input:        `"hello\u0041world"`,
			expectedText: "\"helloAworld\"",
			expectError:  false,
		},
		{
			name:         "unicode escape hex lowercase",
			input:        `"\u00e9"`,
			expectedText: "\"é\"",
			expectError:  false,
		},
		{
			name:         "unicode escape hex uppercase",
			input:        `"\u00C9"`,
			expectedText: "\"É\"",
			expectError:  false,
		},
		{
			name:        "invalid escape sequence",
			input:       `"hello\xworld"`,
			expectError: true,
		},
		{
			name:        "invalid unicode escape - not hex",
			input:       `"hello\u00gworld"`,
			expectError: true,
		},
		{
			name:        "invalid unicode escape - too short",
			input:       `"hello\u00"`,
			expectError: true,
		},
		{
			name:        "unterminated string",
			input:       `"hello`,
			expectError: true,
		},
		{
			name:        "unterminated string with newline",
			input:       "\"hello\n",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tc.input))
			tok, err := scanner.Scan()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tok.Type != TokenError {
					t.Errorf("Expected TokenError, got %q", tok.Type)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if tok.Type != TokenString {
					t.Errorf("Expected TokenString, got %q", tok.Type)
				}
				if tok.Text != tc.expectedText {
					t.Errorf("Expected text %q, got %q", tc.expectedText, tok.Text)
				}
			}
		})
	}
}

func TestScanComments(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectedText string
	}{
		{
			name:         "line comment",
			input:        "// this is a comment",
			expectedText: "// this is a comment",
		},
		{
			name:         "line comment with content after",
			input:        "// comment\nsyntax",
			expectedText: "// comment",
		},
		{
			name:         "block comment single line",
			input:        "/* comment */",
			expectedText: "/* comment */",
		},
		{
			name:         "block comment multi line",
			input:        "/* line1\nline2\nline3 */",
			expectedText: "/* line1\nline2\nline3 */",
		},
		{
			name:         "block comment with stars",
			input:        "/* * stars * */",
			expectedText: "/* * stars * */",
		},
		{
			name:         "unterminated block comment",
			input:        "/* unterminated",
			expectedText: "/* unterminated",
		},
		{
			name:         "block comment with multiple closing attempts",
			input:        "/* test * * * */",
			expectedText: "/* test * * * */",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tc.input))
			tok, err := scanner.Scan()

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tok.Type != TokenComment {
				t.Errorf("Expected TokenComment, got %q", tok.Type)
			}

			if tok.Text != tc.expectedText {
				t.Errorf("Expected text %q, got %q", tc.expectedText, tok.Text)
			}
		})
	}
}

func TestScanForwardSlashNotComment(t *testing.T) {
	// Test that a single forward slash not followed by / or * is treated as punctuation
	input := "/ "
	scanner := NewScanner(strings.NewReader(input))
	tok, err := scanner.Scan()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if tok.Type != TokenPunctuation {
		t.Errorf("Expected TokenPunctuation, got %q", tok.Type)
	}

	if tok.Text != "/" {
		t.Errorf("Expected text %q, got %q", "/", tok.Text)
	}
}

func TestScanErrorCases(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid character",
			input: "@",
		},
		{
			name:  "invalid character with context",
			input: "syntax @ proto3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tc.input))

			// Scan until we hit the error
			for {
				tok, err := scanner.Scan()
				if tok.Type == TokenEOF {
					t.Fatal("Expected error token but reached EOF")
				}
				if tok.Type == TokenError {
					if err == nil {
						t.Error("Expected error to be returned with TokenError")
					}
					break
				}
				if err != nil {
					break
				}
			}
		})
	}
}

func TestScanComplexProto(t *testing.T) {
	input := `syntax = "proto3";
package example;

// User message
message User {
  string id = 1;
  int32 age = 2;
}`

	scanner := NewScanner(strings.NewReader(input))

	expectedTokens := []struct {
		tokenType TokenType
		text      string
	}{
		{TokenIdentifier, "syntax"},
		{TokenPunctuation, "="},
		{TokenString, `"proto3"`},
		{TokenPunctuation, ";"},
		{TokenIdentifier, "package"},
		{TokenIdentifier, "example"},
		{TokenPunctuation, ";"},
		{TokenComment, "// User message"},
		{TokenIdentifier, "message"},
		{TokenIdentifier, "User"},
		{TokenPunctuation, "{"},
		{TokenIdentifier, "string"},
		{TokenIdentifier, "id"},
		{TokenPunctuation, "="},
		{TokenNumber, "1"},
		{TokenPunctuation, ";"},
		{TokenIdentifier, "int32"},
		{TokenIdentifier, "age"},
		{TokenPunctuation, "="},
		{TokenNumber, "2"},
		{TokenPunctuation, ";"},
		{TokenPunctuation, "}"},
		{TokenEOF, ""},
	}

	for i, expected := range expectedTokens {
		tok, err := scanner.Scan()
		if err != nil {
			t.Fatalf("Token %d: unexpected error: %v", i, err)
		}

		if tok.Type != expected.tokenType {
			t.Errorf("Token %d: expected type %q, got %q (text: %q)", i, expected.tokenType, tok.Type, tok.Text)
		}

		if tok.Text != expected.text {
			t.Errorf("Token %d: expected text %q, got %q", i, expected.text, tok.Text)
		}
	}
}

func TestScanWhitespace(t *testing.T) {
	input := "  \t\n  syntax  \n\t  "
	scanner := NewScanner(strings.NewReader(input))

	tok, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should skip all leading whitespace and get 'syntax'
	if tok.Type != TokenIdentifier {
		t.Errorf("Expected TokenIdentifier, got %q", tok.Type)
	}

	if tok.Text != "syntax" {
		t.Errorf("Expected text %q, got %q", "syntax", tok.Text)
	}
}

func TestScanPosition(t *testing.T) {
	input := "syntax\n  id"
	scanner := NewScanner(strings.NewReader(input))

	// First token: 'syntax' on line 1
	tok, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if tok.Pos.Line != 1 {
		t.Errorf("Expected line 1, got %d", tok.Pos.Line)
	}

	// Second token: 'id' on line 2
	tok, err = scanner.Scan()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if tok.Pos.Line != 2 {
		t.Errorf("Expected line 2, got %d", tok.Pos.Line)
	}
}

func TestReadString(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		delimiter   rune
		expected    string
		expectError bool
	}{
		{
			name:        "simple string",
			input:       `"hello"`,
			delimiter:   '"',
			expected:    "hello",
			expectError: false,
		},
		{
			name:        "string with newline escape",
			input:       `"hello\nworld"`,
			delimiter:   '"',
			expected:    "hello\nworld",
			expectError: false,
		},
		{
			name:        "string with tab escape",
			input:       `"hello\tworld"`,
			delimiter:   '"',
			expected:    "hello\tworld",
			expectError: false,
		},
		{
			name:        "string with carriage return escape",
			input:       `"hello\rworld"`,
			delimiter:   '"',
			expected:    "hello\rworld",
			expectError: false,
		},
		{
			name:        "string with backslash escape",
			input:       `"hello\\world"`,
			delimiter:   '"',
			expected:    "hello\\world",
			expectError: false,
		},
		{
			name:        "string with quote escape",
			input:       `"hello\"world"`,
			delimiter:   '"',
			expected:    "hello\"world",
			expectError: false,
		},
		{
			name:        "single quote string",
			input:       `'hello'`,
			delimiter:   '\'',
			expected:    "hello",
			expectError: false,
		},
		{
			name:        "invalid escape sequence",
			input:       `"hello\xworld"`,
			delimiter:   '"',
			expectError: true,
		},
		{
			name:        "wrong starting delimiter",
			input:       `'hello"`,
			delimiter:   '"',
			expectError: true,
		},
		{
			name:        "EOF before closing delimiter",
			input:       `"hello`,
			delimiter:   '"',
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tc.input))
			result, err := readString(reader, tc.delimiter)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if result != tc.expected {
					t.Errorf("Expected %q, got %q", tc.expected, result)
				}
			}
		})
	}
}

func TestIsIdentifier(t *testing.T) {
	testCases := []struct {
		r        rune
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'_', true},
		{'.', true},
		{'0', true},
		{'9', true},
		{'-', false},
		{'@', false},
		{' ', false},
		{'\n', false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.r), func(t *testing.T) {
			result := isIdentifier(tc.r)
			if result != tc.expected {
				t.Errorf("isIdentifier(%q): expected %v, got %v", tc.r, tc.expected, result)
			}
		})
	}
}

func TestIsNumber(t *testing.T) {
	testCases := []struct {
		r        rune
		expected bool
	}{
		{'0', true},
		{'9', true},
		{'.', true},
		{'+', true},
		{'-', true},
		{'e', true},
		{'E', true},
		{'a', false},
		{'x', false},
		{'_', false},
		{' ', false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.r), func(t *testing.T) {
			result := isNumber(tc.r)
			if result != tc.expected {
				t.Errorf("isNumber(%q): expected %v, got %v", tc.r, tc.expected, result)
			}
		})
	}
}

func TestIsPunctuation(t *testing.T) {
	testCases := []struct {
		r        rune
		expected bool
	}{
		{';', true},
		{',', true},
		{'=', true},
		{'{', true},
		{'}', true},
		{'[', true},
		{']', true},
		{'(', true},
		{')', true},
		{'<', true},
		{'>', true},
		{':', true},
		{'a', false},
		{'0', false},
		{' ', false},
		{'/', false},
		{'@', false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.r), func(t *testing.T) {
			result := isPunctuation(tc.r)
			if result != tc.expected {
				t.Errorf("isPunctuation(%q): expected %v, got %v", tc.r, tc.expected, result)
			}
		})
	}
}

func TestRuneToString(t *testing.T) {
	testCases := []struct {
		name     string
		r        rune
		expected string
	}{
		{"ascii letter", 'a', "a"},
		{"ascii digit", '5', "5"},
		{"unicode", '€', "€"},
		{"emoji", '😀', "😀"},
		{"chinese", '中', "中"},
		{"newline", '\n', "\n"},
		{"tab", '\t', "\t"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := runeToString(tc.r)
			if result != tc.expected {
				t.Errorf("runeToString(%q): expected %q, got %q", tc.r, tc.expected, result)
			}
		})
	}
}

func TestPeek(t *testing.T) {
	input := "abc"
	scanner := NewScanner(strings.NewReader(input))

	// Peek at next character (should be 'b' since 'a' was consumed during init)
	next, err := scanner.peek()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// peek should return the next character without advancing
	if next != 'b' {
		t.Errorf("Expected 'b', got %q", next)
	}

	// Current character should still be 'a'
	if scanner.ch != 'a' {
		t.Errorf("Expected current char 'a', got %q", scanner.ch)
	}

	// After next(), should get 'b'
	scanner.next()
	if scanner.ch != 'b' {
		t.Errorf("Expected current char 'b' after next(), got %q", scanner.ch)
	}
}

func TestPeekAtEOF(t *testing.T) {
	input := ""
	scanner := NewScanner(strings.NewReader(input))

	// Should be at EOF already
	if scanner.ch != -1 {
		t.Errorf("Expected EOF (-1), got %v", scanner.ch)
	}

	// Peek should return error at EOF
	_, err := scanner.peek()
	if err == nil {
		t.Error("Expected error when peeking at EOF")
	}
}

func TestScannerLineAndColumn(t *testing.T) {
	input := "a\nb\nc"
	scanner := NewScanner(strings.NewReader(input))

	// Initial position (after reading 'a')
	if scanner.line != 1 {
		t.Errorf("Expected line 1, got %d", scanner.line)
	}

	// Move to newline
	scanner.next()
	if scanner.line != 2 {
		t.Errorf("After newline, expected line 2, got %d", scanner.line)
	}

	if scanner.column != 1 {
		t.Errorf("After newline, expected column 1, got %d", scanner.column)
	}

	// Move to 'b'
	scanner.next()
	if scanner.line != 2 {
		t.Errorf("Expected line 2, got %d", scanner.line)
	}

	if scanner.column != 2 {
		t.Errorf("Expected column 2, got %d", scanner.column)
	}
}

func TestScanMultipleStrings(t *testing.T) {
	input := `"first" "second"`
	scanner := NewScanner(strings.NewReader(input))

	tok1, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Unexpected error on first string: %v", err)
	}
	if tok1.Type != TokenString {
		t.Errorf("Expected TokenString, got %q", tok1.Type)
	}
	if tok1.Text != `"first"` {
		t.Errorf("Expected %q, got %q", `"first"`, tok1.Text)
	}

	tok2, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Unexpected error on second string: %v", err)
	}
	if tok2.Type != TokenString {
		t.Errorf("Expected TokenString, got %q", tok2.Type)
	}
	if tok2.Text != `"second"` {
		t.Errorf("Expected %q, got %q", `"second"`, tok2.Text)
	}
}

func TestScanEmptyString(t *testing.T) {
	input := `""`
	scanner := NewScanner(strings.NewReader(input))

	tok, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if tok.Type != TokenString {
		t.Errorf("Expected TokenString, got %q", tok.Type)
	}

	if tok.Text != `""` {
		t.Errorf("Expected empty string %q, got %q", `""`, tok.Text)
	}
}

func TestScanNumberVariations(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"integer", "42"},
		{"float", "3.14159"},
		{"scientific lowercase", "1.23e10"},
		{"scientific uppercase", "1.23E10"},
		{"scientific negative", "1.5e-3"},
		{"scientific positive", "1.5e+3"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scanner := NewScanner(strings.NewReader(tc.input))
			tok, err := scanner.Scan()

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tok.Type != TokenNumber {
				t.Errorf("Expected TokenNumber, got %q", tok.Type)
			}

			if tok.Text != tc.input {
				t.Errorf("Expected %q, got %q", tc.input, tok.Text)
			}
		})
	}
}

func TestLastToken(t *testing.T) {
	input := "syntax = \"proto3\";"
	scanner := NewScanner(strings.NewReader(input))

	// Scan first token
	tok1, _ := scanner.Scan()
	if scanner.lastToken != tok1.Type {
		t.Errorf("Expected lastToken to be %q, got %q", tok1.Type, scanner.lastToken)
	}

	// Scan second token
	tok2, _ := scanner.Scan()
	if scanner.lastToken != tok2.Type {
		t.Errorf("Expected lastToken to be %q, got %q", tok2.Type, scanner.lastToken)
	}
}
