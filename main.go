// main.go
package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

const defaultLen = 16
const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: goki <KeyName> <./path_to_keys.go>")
		os.Exit(2)
	}

	keyName := os.Args[1]
	outPath := os.Args[2]

	if !token.IsIdentifier(keyName) {
		fmt.Fprintf(os.Stderr, "Error: %q is not a valid Go identifier\n", keyName)
		os.Exit(2)
	}

	key, err := randString(defaultLen)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error generating key:", err)
		os.Exit(1)
	}

	if err := writeOrAppendKey(outPath, keyName, key); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s to %s\n", keyName, outPath)
}

func writeOrAppendKey(outPath, keyName, key string) error {
	dir := filepath.Dir(outPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	line := fmt.Sprintf("const %s = %q\n", keyName, key)

	_, err := os.Stat(outPath)
	switch {
	case err == nil:
		// File exists: verify it’s a keys package file and ensure key doesn't already exist.
		b, readErr := os.ReadFile(outPath)
		if readErr != nil {
			return readErr
		}
		src := string(b)

		if !looksLikeKeysPackage(src) {
			return errors.New("existing file does not appear to declare `package keys`")
		}
		if keyExists(src, keyName) {
			return fmt.Errorf("key %q already exists in %s", keyName, outPath)
		}

		f, openErr := os.OpenFile(outPath, os.O_APPEND|os.O_WRONLY, 0)
		if openErr != nil {
			return openErr
		}
		defer f.Close()

		// Ensure newline separation before appending.
		if len(b) > 0 && b[len(b)-1] != '\n' {
			if _, werr := f.WriteString("\n"); werr != nil {
				return werr
			}
		}
		if len(strings.TrimSpace(src)) > 0 {
			if _, werr := f.WriteString("\n"); werr != nil {
				return werr
			}
		}

		_, werr := f.WriteString(line)
		return werr

	case os.IsNotExist(err):
		// File missing: create it.
		content := "package keys\n\n" + line
		return os.WriteFile(outPath, []byte(content), 0o644)

	default:
		return err
	}
}

func looksLikeKeysPackage(src string) bool {
	for _, ln := range strings.Split(src, "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "package ") {
			return t == "package keys"
		}
	}
	return false
}

func keyExists(src, keyName string) bool {
	for _, ln := range strings.Split(src, "\n") {
		t := strings.TrimSpace(ln)
		if !strings.HasPrefix(t, "const ") {
			continue
		}
		// Handles: const Name = "..."
		fields := strings.Fields(t)
		if len(fields) >= 2 && fields[1] == keyName {
			return true
		}
	}
	return false
}

func randString(n int) (string, error) {
	if n <= 0 {
		return "", errors.New("length must be > 0")
	}

	alphabetLen := byte(len(alphabet))
	maxMultiple := byte(256 / int(alphabetLen) * int(alphabetLen))

	out := make([]byte, 0, n)
	buf := make([]byte, n*2)

	for len(out) < n {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if b >= maxMultiple {
				continue
			}
			out = append(out, alphabet[b%alphabetLen])
			if len(out) == n {
				break
			}
		}
	}

	return string(out), nil
}
