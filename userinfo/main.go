package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"

	srvConfig "github.com/CHESSComputing/golib/config"
	"github.com/CHESSComputing/golib/ldap"
)

// Patterns used to classify the positional argument.
var (
	reUidNumber = regexp.MustCompile(`^\d+$`)              // pure integer  → uidNumber
	reEmail     = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)  // contains @    → email
	reUid       = regexp.MustCompile(`^[a-zA-Z]{1,3}\d+$`) // ≤3 letters + digits → uid
	// anything else is treated as a display name
)

var ldapCache *ldap.Cache

func main() {
	var (
		baseURL = flag.String("url", "", "URL of ClasseInfoService")
		gid     = flag.String("gid", "", "Lookup by GID number")
		timeout = flag.Duration("timeout", 10*time.Second, "HTTP request timeout")
		jsonOut = flag.Bool("json", false, "Output raw JSON instead of formatted display")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n\n", colorize(bold+cyan, "userinfo — CLASSE User Service CLI"))
		fmt.Fprintf(os.Stderr, "%s\n", colorize(bold, "USAGE"))
		fmt.Fprintf(os.Stderr, "  userinfo [options] <value>\n\n")
		fmt.Fprintf(os.Stderr, "%s\n", colorize(bold, "ARGUMENT (auto-detected)"))
		fmt.Fprintf(os.Stderr, "  %-24s  %s\n", "<number>", "Lookup by uidNumber  (e.g. 123)")
		fmt.Fprintf(os.Stderr, "  %-24s  %s\n", "<user>@<domain>", "Lookup by email      (e.g. user@cornell.edu)")
		fmt.Fprintf(os.Stderr, "  %-24s  %s\n", "<≤3 letters + digits>", "Lookup by uid        (e.g. abc1, xyz12)")
		fmt.Fprintf(os.Stderr, "  %-24s  %s\n", "<anything else>", "Lookup by name       (e.g. \"Jane Smith\")")
		fmt.Fprintf(os.Stderr, "\n%s\n", colorize(bold, "OPTIONS"))
		fmt.Fprintf(os.Stderr, "  %-24s  %s\n", "-gid <number>", "Lookup by GID number")
		fmt.Fprintf(os.Stderr, "  %-24s  %s\n", "-url <string>", "Base URL (default: http://host-dev:8080)")
		fmt.Fprintf(os.Stderr, "  %-24s  %s\n", "-timeout <duration>", "Request timeout (default: 10s)")
		fmt.Fprintf(os.Stderr, "  %-24s  %s\n", "-json", "Print raw JSON output")
		fmt.Fprintf(os.Stderr, "\n%s\n", colorize(bold, "EXAMPLES"))
		fmt.Fprintf(os.Stderr, "  userinfo 123\n")
		fmt.Fprintf(os.Stderr, "  userinfo abc1\n")
		fmt.Fprintf(os.Stderr, "  userinfo user@cornell.edu\n")
		fmt.Fprintf(os.Stderr, "  userinfo \"Jane Smith\"\n")
		fmt.Fprintf(os.Stderr, "  userinfo -gid 42\n")
		fmt.Fprintf(os.Stderr, "  userinfo -json 123\n")
	}

	flag.Parse()

	// parse FOXDEN config
	config := os.Getenv("FOXDEN_CONFIG")
	if cobj, err := srvConfig.ParseConfig(config); err == nil {
		srvConfig.Config = &cobj
	}

	// initialize ldap cache
	ldapCache = &ldap.Cache{Map: make(map[string]ldap.Entry)}

	var paramKey, paramValue string

	userInput := UserInput{}
	if *gid != "" {
		// Explicit -gid flag takes priority
		if !reUidNumber.MatchString(*gid) {
			fatalf("-gid value must be a number, got %q", *gid)
		}
		paramKey, paramValue = "gidNumber", *gid
		userInput.GidNumber = *gid
	} else {
		args := flag.Args()
		if len(args) == 0 {
			flag.Usage()
			os.Exit(1)
		}
		if len(args) > 1 {
			fatalf("expected a single argument, got %d — wrap multi-word names in quotes", len(args))
		}
		paramKey, paramValue = classifyArg(args[0])
		switch paramKey {
		case "name":
			userInput.Name = paramValue
		case "email":
			userInput.Email = paramValue
		case "uid":
			userInput.Uid = paramValue
		case "UidNumber":
			userInput.UidNumber = paramValue

		}
	}

	var users []UserInfo
	var err error
	if *baseURL != "" {
		targetURL, err := buildURL(*baseURL, paramKey, paramValue)
		if err != nil {
			fatalf("%v", err)
		}
		users, err = fetchUsers(targetURL, *timeout)
		if err != nil {
			fatalf("%v", err)
		}
	} else {
		users, err = userLookup(userInput)
		if err != nil {
			fatalf("%v", err)
		}
	}

	if len(users) == 0 {
		fmt.Println(colorize(yellow, "No users found matching your query."))
		os.Exit(0)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(users)
		return
	}

	fmt.Println()
	for _, u := range users {
		printUser(u)
		fmt.Println()
	}

	if len(users) > 1 {
		fmt.Printf(colorize(dim, "  %d record(s) returned.\n\n"), len(users))
	}
}
