package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/CHESSComputing/DataBookkeeping/dbs"
)

// ProvenanceParameters holds all parameters we may need to generate provenance record
type ProvenanceParameters struct {
	Did               string
	App               string
	ConfigFile        string
	InputDir          string
	InputFilePattern  string
	OutputDir         string
	OutputFilePattern string
	InputFileList     string
	OutputFileList    string
}

// helper function to generate provenance record
func generateProvenanceRecord(rec ProvenanceParameters) {

	content, _ := ReadFileContent(rec.ConfigFile)
	config := dbs.ConfigRecord{Content: content}

	var inputFiles, outputFiles []dbs.FileRecord
	if content, err := ReadFileContent(rec.InputFileList); err == nil {
		for f := range strings.SplitSeq(content, "\n") {
			if f != "" {
				size, chksum, _ := FileInfo(f)
				rec := dbs.FileRecord{Name: f, Size: size, Checksum: chksum}
				inputFiles = append(inputFiles, rec)
			}
		}
	}
	if content, err := ReadFileContent(rec.OutputFileList); err == nil {
		for f := range strings.SplitSeq(content, "\n") {
			if f != "" {
				size, chksum, _ := FileInfo(f)
				rec := dbs.FileRecord{Name: f, Size: size, Checksum: chksum}
				outputFiles = append(outputFiles, rec)
			}
		}
	}
	for _, f := range FileList(rec.InputDir, rec.InputFilePattern) {
		if f != "" {
			size, chksum, _ := FileInfo(f)
			rec := dbs.FileRecord{Name: f, Size: size, Checksum: chksum}
			inputFiles = append(inputFiles, rec)
		}
	}
	for _, f := range FileList(rec.OutputDir, rec.OutputFilePattern) {
		if f != "" {
			size, chksum, _ := FileInfo(f)
			rec := dbs.FileRecord{Name: f, Size: size, Checksum: chksum}
			outputFiles = append(outputFiles, rec)
		}
	}

	var envs []dbs.EnvironmentRecord
	env, _ := CreateEnvironmentRecord()
	envs = append(envs, env)

	var scripts []dbs.ScriptRecord
	srec := dbs.ScriptRecord{Name: rec.App}
	scripts = append(scripts, srec)
	osInfo, _ := GetOsInfo()

	prov := dbs.ProvenanceRecord{
		Did: rec.Did, Site: "Cornell", Processing: rec.App, Config: config,
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

// FileInfo return size and checksum of the file
func FileInfo(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return 0, "", err
	}
	size := stat.Size()

	// Compute checksum
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return 0, "", err
	}
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	return size, checksum, nil
}

// FileList walks through the directory tree starting at dir
// and returns a slice of full paths of files matching the pattern pat.
func FileList(dir string, pat string) []string {
	var files []string

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// skip files/directories we cannot access
			return nil
		}

		if !info.IsDir() {
			if pat != "" {
				match, err := filepath.Match(pat, info.Name())
				if err != nil {
					// skip invalid patterns
					return nil
				}
				if match {
					files = append(files, path)
				}
			} else {
				// if no pattern is provided we will take file path as is
				files = append(files, path)
			}
		}
		return nil
	})

	return files
}

// ReadFileContent reads the content of the given file.
// If the file does not exist, it returns an empty string and no error.
// If the file exists, it returns its contents as a string.
func ReadFileContent(fname string) (string, error) {
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
