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

// ANSI color/style codes
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	cyan    = "\033[36m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	magenta = "\033[35m"
	red     = "\033[31m"
	bgDark  = "\033[48;5;236m"
	white   = "\033[97m"
)

func colorize(code, s string) string {
	return code + s + reset
}

func printDivider(width int, char string) {
	fmt.Println(colorize(dim+cyan, strings.Repeat(char, width)))
}

func printField(label, value string) {
	if value == "" || value == "0" {
		return
	}
	fmt.Printf("  %s%-14s%s %s\n",
		colorize(bold+yellow, ""),
		colorize(bold+yellow, label+":"),
		reset,
		colorize(white, value),
	)
}

func printListField(label string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("  %s%-14s%s\n", colorize(bold+yellow, ""), colorize(bold+yellow, label+":"), reset)
	for _, item := range items {
		// For DN-style entries, show just the CN part for readability, with full path dimmed
		if strings.Contains(item, "CN=") && strings.Contains(item, ",") {
			parts := strings.SplitN(item, ",", 2)
			cn := strings.TrimPrefix(parts[0], "CN=")
			rest := "," + parts[1]
			fmt.Printf("    %s %s%s\n",
				colorize(bold+green, "▸"),
				colorize(white, cn),
				colorize(dim, rest),
			)
		} else {
			fmt.Printf("    %s %s\n", colorize(bold+green, "▸"), colorize(white, item))
		}
	}
}

func printExpiry(t time.Time) {
	now := time.Now()
	diff := t.Sub(now)

	var status string
	switch {
	case t.IsZero():
		return
	case diff < 0:
		status = colorize(bold+red, "EXPIRED "+t.Format("2006-01-02"))
	case diff < 30*24*time.Hour:
		days := int(diff.Hours() / 24)
		status = colorize(bold+yellow, fmt.Sprintf("expires in %d days (%s)", days, t.Format("2006-01-02")))
	default:
		status = colorize(green, t.Format("2006-01-02 15:04:05 MST"))
	}

	fmt.Printf("  %s%-14s%s %s\n",
		colorize(bold+yellow, ""),
		colorize(bold+yellow, "Expire:"),
		reset,
		status,
	)
}

func printUser(u UserInfo) {
	width := 60
	printDivider(width, "─")

	// Header with name
	header := fmt.Sprintf(" 👤  %s", u.Name)
	fmt.Printf("%s%s%s%s\n",
		colorize(bold+bgDark+cyan, header),
		strings.Repeat(" ", max(0, width-len(header)-1)),
		colorize(bgDark, " "),
		reset,
	)

	printDivider(width, "─")
	fmt.Println()

	// Identity block
	fmt.Println(colorize(bold+magenta, "  IDENTITY"))
	printField("UID", u.Uid)
	if u.UidNumber != 0 {
		printField("UID Number", fmt.Sprintf("%d", u.UidNumber))
	}
	if u.GidNumber != 0 {
		printField("GID Number", fmt.Sprintf("%d", u.GidNumber))
	}
	printField("Email", u.Email)
	fmt.Println()

	// Directory
	if u.DN != "" {
		fmt.Println(colorize(bold+magenta, "  DIRECTORY"))
		fmt.Printf("  %s%-14s%s %s\n",
			colorize(bold+yellow, ""),
			colorize(bold+yellow, "DN:"),
			reset,
			colorize(dim+white, u.DN),
		)
		fmt.Println()
	}

	// Access block
	fmt.Println(colorize(bold+magenta, "  ACCESS"))
	printListField("Beamlines", u.Beamlines)
	printListField("BTRs", u.Btrs)
	printListField("Foxdens", u.Foxdens)
	fmt.Println()

	// Groups
	if len(u.Groups) > 0 {
		fmt.Println(colorize(bold+magenta, "  GROUPS"))
		printListField("", u.Groups)
		fmt.Println()
	}

	// Expiry
	fmt.Println(colorize(bold+magenta, "  ACCOUNT STATUS"))
	printExpiry(u.Expire)
	fmt.Println()

	printDivider(width, "─")
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
	fmt.Fprintf(os.Stderr, colorize(bold+red, "error")+": "+format+"\n", args...)
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
