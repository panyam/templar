package templar

// ToolInfo describes the embedding tool for use in generated content,
// error messages, and file discovery. Embedding applications (like slyds)
// provide their own ToolInfo to customize all templar-generated artifacts.
type ToolInfo struct {
	Name        string   // Tool name, e.g. "templar", "slyds"
	ConfigNames []string // Config file names to search for, e.g. ["templar.yaml", ".templar.yaml"]
	VendorDir   string   // Default vendor directory name, e.g. "./templar_modules"
	LockFile    string   // Lock file name, e.g. "templar.lock"
	FetchCmd    string   // Command to fetch dependencies, e.g. "templar get"
	ProjectURL  string   // Project URL for generated content, e.g. "https://github.com/panyam/templar"
}

const (
	// DefaultVendorDir is the default directory for vendored template sources.
	DefaultVendorDir = "./templar_modules"

	// DefaultLockFile is the default lock file name.
	DefaultLockFile = "templar.lock"
)

// DefaultConfigNames is the ordered list of config file names to search for.
var DefaultConfigNames = []string{"templar.yaml", ".templar.yaml"}

// DefaultToolInfo returns a ToolInfo configured with templar's standard defaults.
func DefaultToolInfo() ToolInfo {
	return ToolInfo{
		Name:        "templar",
		ConfigNames: DefaultConfigNames,
		VendorDir:   DefaultVendorDir,
		LockFile:    DefaultLockFile,
		FetchCmd:    "templar get",
		ProjectURL:  "https://github.com/panyam/templar",
	}
}
