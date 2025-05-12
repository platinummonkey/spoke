package protobuf

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenType represents the type of token
type TokenType string

const (
	TokenIdentifier  TokenType = "IDENTIFIER"
	TokenString      TokenType = "STRING"
	TokenNumber      TokenType = "NUMBER"
	TokenPunctuation TokenType = "PUNCTUATION"
	TokenComment     TokenType = "COMMENT"
	TokenEOF         TokenType = "EOF"
	TokenError       TokenType = "ERROR"
)

// Token represents a lexical token
type Token struct {
	Type TokenType
	Text string
	Pos  Position
}

// Scanner represents a lexical scanner for protobuf
type Scanner struct {
	r         *bufio.Reader
	ch        rune     // current character
	offset    int      // character offset
	rdOffset  int      // reading offset (position after current character)
	line      int      // current line number
	column    int      // current column number
	lastToken TokenType // last token type for context
}

// NewScanner creates a new Scanner
func NewScanner(r io.Reader) *Scanner {
	br := bufio.NewReader(r)
	s := &Scanner{
		r:     br,
		line:  1,
		column: 1,
	}
	s.next()
	return s
}

// next reads the next Unicode character into s.ch
// and updates the line/column position
func (s *Scanner) next() {
	r, size, err := s.r.ReadRune()
	if err != nil {
		s.ch = -1 // EOF
		return
	}
	
	s.offset = s.rdOffset
	s.rdOffset += size
	
	// Update line and column counts
	if r == '\n' {
		s.line++
		s.column = 1
	} else {
		s.column++
	}
	
	s.ch = r
}

// peek returns the next rune without advancing
func (s *Scanner) peek() (rune, error) {
	r, _, err := s.r.ReadRune()
	if err != nil {
		return 0, err
	}
	// Put the rune back
	if err := s.r.UnreadRune(); err != nil {
		return 0, err
	}
	return r, nil
}

// skipWhitespace skips whitespace characters
func (s *Scanner) skipWhitespace() {
	for unicode.IsSpace(s.ch) {
		s.next()
	}
}

// scanIdentifier scans an identifier
func (s *Scanner) scanIdentifier() string {
	var sb strings.Builder
	for unicode.IsLetter(s.ch) || unicode.IsDigit(s.ch) || s.ch == '_' || s.ch == '.' {
		sb.WriteRune(s.ch)
		s.next()
	}
	return sb.String()
}

// scanNumber scans a number
func (s *Scanner) scanNumber() string {
	var sb strings.Builder
	for unicode.IsDigit(s.ch) || s.ch == '.' || s.ch == 'e' || s.ch == 'E' || s.ch == '+' || s.ch == '-' {
		sb.WriteRune(s.ch)
		s.next()
	}
	return sb.String()
}

// scanString scans a string
func (s *Scanner) scanString() (string, error) {
	quote := s.ch // ' or "
	s.next()      // consume the opening quote
	
	var sb strings.Builder
	for s.ch != quote && s.ch != -1 {
		if s.ch == '\\' {
			s.next() // consume backslash
			switch s.ch {
			case 'n':
				sb.WriteRune('\n')
			case 'r':
				sb.WriteRune('\r')
			case 't':
				sb.WriteRune('\t')
			case '\\', '"', '\'':
				sb.WriteRune(s.ch)
			case 'u':
				// Handle unicode escape sequences
				s.next() // consume 'u'
				var hexStr strings.Builder
				for i := 0; i < 4; i++ {
					if !((s.ch >= '0' && s.ch <= '9') || (s.ch >= 'a' && s.ch <= 'f') || (s.ch >= 'A' && s.ch <= 'F')) {
						return "", fmt.Errorf("invalid unicode escape sequence")
					}
					hexStr.WriteRune(s.ch)
					s.next()
				}
				// Convert hex to rune
				var val int
				fmt.Sscanf(hexStr.String(), "%x", &val)
				sb.WriteRune(rune(val))
				continue // already consumed the next character
			default:
				return "", fmt.Errorf("invalid escape sequence: \\%c", s.ch)
			}
		} else {
			sb.WriteRune(s.ch)
		}
		s.next()
	}
	
	if s.ch != quote {
		return "", fmt.Errorf("unterminated string")
	}
	s.next() // consume the closing quote
	
	return sb.String(), nil
}

// scanComment scans a comment
func (s *Scanner) scanComment() string {
	var sb strings.Builder
	sb.WriteRune(s.ch) // Add the first '/'
	s.next()
	
	if s.ch == '/' {
		// Line comment
		sb.WriteRune(s.ch) // Add the second '/'
		s.next()
		for s.ch != '\n' && s.ch != -1 {
			sb.WriteRune(s.ch)
			s.next()
		}
	} else if s.ch == '*' {
		// Block comment
		sb.WriteRune(s.ch) // Add the '*'
		s.next()
		for {
			if s.ch == '*' {
				sb.WriteRune(s.ch)
				s.next()
				if s.ch == '/' {
					sb.WriteRune(s.ch)
					s.next()
					break
				}
			} else if s.ch == -1 {
				break // Unterminated comment
			} else {
				sb.WriteRune(s.ch)
				s.next()
			}
		}
	}
	
	return sb.String()
}

// Scan returns the next token
func (s *Scanner) Scan() (Token, error) {
	s.skipWhitespace()
	
	pos := Position{
		Line:   s.line,
		Column: s.column,
		Offset: s.offset,
	}
	
	var tok Token
	tok.Pos = pos
	
	switch {
	case s.ch == -1:
		tok.Type = TokenEOF
		tok.Text = ""
	case unicode.IsLetter(s.ch) || s.ch == '_':
		// Identifier
		text := s.scanIdentifier()
		tok.Type = TokenIdentifier
		tok.Text = text
	case unicode.IsDigit(s.ch):
		// Number
		text := s.scanNumber()
		tok.Type = TokenNumber
		tok.Text = text
	case s.ch == '"' || s.ch == '\'':
		// String
		text, err := s.scanString()
		if err != nil {
			tok.Type = TokenError
			tok.Text = err.Error()
			return tok, err
		}
		tok.Type = TokenString
		tok.Text = "\"" + text + "\""
	case s.ch == '/':
		// Potential comment
		next, err := s.peek()
		if err == nil && (next == '/' || next == '*') {
			text := s.scanComment()
			tok.Type = TokenComment
			tok.Text = text
		} else {
			// Just a forward slash
			tok.Type = TokenPunctuation
			tok.Text = "/"
			s.next()
		}
	case s.ch == ';' || s.ch == ',' || s.ch == '=' || s.ch == '{' || s.ch == '}' || s.ch == '[' || s.ch == ']' || s.ch == '(' || s.ch == ')' || s.ch == '<' || s.ch == '>' || s.ch == ':':
		// Punctuation
		tok.Type = TokenPunctuation
		tok.Text = string(s.ch)
		s.next()
	default:
		// Unknown
		tok.Type = TokenError
		tok.Text = string(s.ch)
		s.next()
		return tok, fmt.Errorf("unexpected character: %c", s.ch)
	}
	
	s.lastToken = tok.Type
	return tok, nil
}

// readString reads a string from the input until the delimiter is reached
func readString(r *bufio.Reader, delimiter rune) (string, error) {
	var sb strings.Builder
	
	// Read the first character (should be the delimiter)
	firstChar, _, err := r.ReadRune()
	if err != nil {
		return "", err
	}
	if firstChar != delimiter {
		return "", fmt.Errorf("expected delimiter: %c", delimiter)
	}
	
	for {
		c, _, err := r.ReadRune()
		if err != nil {
			return "", err
		}
		
		if c == '\\' {
			// Handle escape sequences
			escChar, _, err := r.ReadRune()
			if err != nil {
				return "", err
			}
			
			switch escChar {
			case 'n':
				sb.WriteRune('\n')
			case 'r':
				sb.WriteRune('\r')
			case 't':
				sb.WriteRune('\t')
			case delimiter, '\\':
				sb.WriteRune(escChar)
			default:
				return "", fmt.Errorf("invalid escape sequence: \\%c", escChar)
			}
		} else if c == delimiter {
			break
		} else {
			sb.WriteRune(c)
		}
	}
	
	return sb.String(), nil
}

// isIdentifier checks if a rune is valid in an identifier
func isIdentifier(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
}

// isNumber checks if a rune is valid in a number
func isNumber(r rune) bool {
	return unicode.IsDigit(r) || r == '.' || r == '+' || r == '-' || r == 'e' || r == 'E'
}

// isPunctuation checks if a rune is a punctuation character
func isPunctuation(r rune) bool {
	switch r {
	case ';', ',', '=', '{', '}', '[', ']', '(', ')', '<', '>', ':':
		return true
	default:
		return false
	}
}

// runeToString converts a rune to a string
func runeToString(r rune) string {
	buf := make([]byte, utf8.RuneLen(r))
	utf8.EncodeRune(buf, r)
	return string(buf)
} 