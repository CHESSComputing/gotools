package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/CHESSComputing/DataBookkeeping/dbs"
)

func main() {
	// Standard JWT fields
	var did string
	flag.StringVar(&did, "did", "", "dataset identifier")
	var app string
	flag.StringVar(&app, "app", "", "user application name")
	var configFile string
	flag.StringVar(&configFile, "configFile", "", "configuration file")
	var inputFile string
	flag.StringVar(&inputFile, "inputFile", "", "file with list of input files")
	var outputFile string
	flag.StringVar(&outputFile, "outputFile", "", "file with list of output files")
	flag.Parse()

	content, _ := readFileContent(configFile)
	config := dbs.ConfigRecord{Content: content}

	var inputFiles, outputFiles []dbs.FileRecord
	if content, err := readFileContent(inputFile); err == nil {
		for _, r := range strings.Split(content, "\n") {
			rec := dbs.FileRecord{Name: r}
			inputFiles = append(inputFiles, rec)
		}
	}
	if content, err := readFileContent(outputFile); err == nil {
		for _, r := range strings.Split(content, "\n") {
			rec := dbs.FileRecord{Name: r}
			outputFiles = append(outputFiles, rec)
		}
	}

	var envs []dbs.EnvironmentRecord
	env, _ := CreateEnvironmentRecord()
	envs = append(envs, env)

	var scripts []dbs.ScriptRecord
	srec := dbs.ScriptRecord{Name: app}
	scripts = append(scripts, srec)
	osInfo, _ := GetOsInfo()

	prov := dbs.ProvenanceRecord{
		Did: did, Site: "Cornell", Processing: app, Config: config,
		InputFiles: inputFiles, OutputFiles: outputFiles,
		Environments: envs, Scripts: scripts, OsInfo: osInfo,
	}
	data, err := json.MarshalIndent(prov, "", "   ")
	if err != nil {
		fmt.Println("ERROR: unable to generate provenance record", err)
		os.Exit(1)
	}
	fmt.Println(string(data))

}

// readFileContent reads the content of the given file.
// If the file does not exist, it returns an empty string and no error.
// If the file exists, it returns its contents as a string.
func readFileContent(fname string) (string, error) {
	// Check if file exists
	_, err := os.Stat(fname)
	if os.IsNotExist(err) {
		// File not found → return empty config
		return "", nil
	} else if err != nil {
		// Other error (e.g., permission denied)
		return "", err
	}

	// File exists → read content
	data, err := os.ReadFile(fname)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetOsInfo collects OS name, version, and kernel info from the current system.
func GetOsInfo() (dbs.OsInfoRecord, error) {
	info := dbs.OsInfoRecord{}

	// OS name
	info.Name = runtime.GOOS

	// Kernel (from `uname -r` if available)
	kernel, err := exec.Command("uname", "-r").Output()
	if err == nil {
		info.Kernel = strings.TrimSpace(string(kernel))
	} else {
		// fallback: runtime version
		info.Kernel = runtime.GOARCH
	}

	// Version
	switch runtime.GOOS {
	case "linux":
		// try /etc/os-release
		data, err := exec.Command("cat", "/etc/os-release").Output()
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					v := strings.TrimPrefix(line, "PRETTY_NAME=")
					info.Version = strings.Trim(v, `"`)
					break
				}
			}
		}
	case "darwin":
		// macOS version
		data, err := exec.Command("sw_vers", "-productVersion").Output()
		if err == nil {
			info.Version = strings.TrimSpace(string(data))
		}
	case "windows":
		// Windows: use wmic if available
		data, err := exec.Command("wmic", "os", "get", "Caption,CSDVersion", "/value").Output()
		if err == nil {
			lines := bytes.Split(data, []byte("\n"))
			for _, line := range lines {
				if bytes.HasPrefix(line, []byte("Caption=")) {
					info.Version = strings.TrimSpace(string(bytes.TrimPrefix(line, []byte("Caption="))))
					break
				}
			}
		}
	}

	if info.Version == "" {
		info.Version = "unknown"
	}
	if info.Kernel == "" {
		info.Kernel = "unknown"
	}

	return info, nil
}

// CreateEnvironmentRecord builds EnvironmentRecord from the current OS and shell environment
func CreateEnvironmentRecord() (dbs.EnvironmentRecord, error) {
	env := dbs.EnvironmentRecord{}

	// --- OS Info ---
	osInfo, err := GetOsInfo()
	if err != nil {
		return env, err
	}
	env.OSName = osInfo.Name
	env.Version = osInfo.Version
	env.Details = osInfo.Kernel

	// --- Shell Info ---
	shell := os.Getenv("SHELL")
	if shell == "" && runtime.GOOS == "windows" {
		shell = os.Getenv("ComSpec") // typical Windows shell (cmd.exe / powershell)
	}
	env.Name = shell

	// --- Parent Process Info ---
	ppid := os.Getppid()
	psCmd := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(ppid))
	if out, err := psCmd.Output(); err == nil {
		env.Parent = strings.TrimSpace(string(out))
	} else {
		env.Parent = "unknown"
	}

	// --- Packages (placeholder: empty for now) ---
	// Here you could add detection of installed packages (pip freeze, npm list, etc.)
	env.Packages = []dbs.PackageRecord{}

	return env, nil
}
