package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func printDivider(width int, char string) {
	fmt.Println(strings.Repeat(char, width))
}

func printField(label, value string) {
	if value == "" || value == "0" {
		return
	}
	fmt.Printf("%14s: %s\n", label, value)
}

func printListField(label string, items []string) {
	if len(items) == 0 {
		return
	}
	if label != "" {
		fmt.Printf("%14s:\n", label)
	}
	for _, item := range items {
		fmt.Printf("  %s\n", item)
	}
}

func printUser(u UserInfo) {
	width := 60
	printDivider(width, "─")
	fmt.Printf(fmt.Sprintf(" 👤  %s\n", u.Name))
	printDivider(width, "─")
	fmt.Printf("UID       : %s\n", u.Uid)
	fmt.Printf("UID Number: %d\n", u.UidNumber)
	fmt.Printf("GID Number: %d\n", u.GidNumber)
	fmt.Printf("Email     : %s\n", u.Email)
	fmt.Printf("DN        : %s\n", u.DN)
	fmt.Println()

	fmt.Printf("Beamlines :\n")
	printListField("", u.Beamlines)
	fmt.Printf("BTRs      :\n")
	printListField("", u.Btrs)
	fmt.Printf("Foxdens   :\n")
	printListField("", u.Foxdens)
	fmt.Println()

	// Groups
	if len(u.Groups) > 0 {
		fmt.Printf("GROUPS    :\n")
		printListField("", u.Groups)
		fmt.Println()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func buildURL(base, paramKey, paramValue string) (string, error) {
	u, err := url.Parse(base + "/translate")
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	q := url.Values{}
	q.Set(paramKey, paramValue)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func fetchUsers(rawURL string, timeout time.Duration) ([]UserInfo, error) {
	client := &http.Client{Timeout: timeout}

	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var users []UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return users, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

// classifyArg returns the query-param key and value to use for the given input.
func classifyArg(arg string) (key, value string) {
	switch {
	case reUidNumber.MatchString(arg):
		return "uidNumber", arg
	case reEmail.MatchString(arg):
		return "email", arg
	case reUid.MatchString(arg):
		return "uid", arg
	default:
		return "name", arg
	}
}
