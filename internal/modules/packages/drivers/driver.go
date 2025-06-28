package drivers

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// PackageDriver defines the interface for package manager implementations
type PackageDriver interface {
	// Name returns the name of the package manager (e.g., "chocolatey", "winget")
	Name() string

	// IsAvailable checks if the package manager is available on the system
	IsAvailable() bool

	// IsPackageInstalled checks if a specific package is installed
	IsPackageInstalled(packageName string) (bool, error)

	// InstallPackage installs a package
	InstallPackage(packageName string) error

	// UninstallPackage uninstalls a package
	UninstallPackage(packageName string) error

	// SearchPackage searches for a package (optional, for future use)
	SearchPackage(packageName string) ([]string, error)

	// GetPackageInfo gets information about an installed package (optional, for future use)
	GetPackageInfo(packageName string) (map[string]string, error)

	// GetAllInstalledPackages returns a map of all installed packages
	GetAllInstalledPackages() (map[string]bool, error)
}

// BaseDriver provides common functionality for all package drivers
type BaseDriver struct {
	name       string
	executable string
	cache      *PackageCache
}

// PackageCache manages cached package information
type PackageCache struct {
	installedPackages map[string]bool
	lastUpdated       time.Time
	cacheDuration     time.Duration
	mutex             sync.RWMutex
}

// NewPackageCache creates a new package cache
func NewPackageCache(duration time.Duration) *PackageCache {
	return &PackageCache{
		installedPackages: make(map[string]bool),
		cacheDuration:     duration,
	}
}

// IsValid checks if the cache is still valid
func (c *PackageCache) IsValid() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return time.Since(c.lastUpdated) < c.cacheDuration
}

// GetPackage returns whether a package is installed from cache
func (c *PackageCache) GetPackage(packageName string) (bool, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	installed, exists := c.installedPackages[packageName]
	return installed, exists
}

// SetPackages updates the entire package cache
func (c *PackageCache) SetPackages(packages map[string]bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.installedPackages = packages
	c.lastUpdated = time.Now()
}

// InvalidateCache clears the cache
func (c *PackageCache) InvalidateCache() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.installedPackages = make(map[string]bool)
	c.lastUpdated = time.Time{}
}

// NewBaseDriver creates a new base driver
func NewBaseDriver(name, executable string) *BaseDriver {
	return &BaseDriver{
		name:       name,
		executable: executable,
		cache:      NewPackageCache(5 * time.Minute), // Cache for 5 minutes
	}
}

// Name returns the driver name
func (d *BaseDriver) Name() string {
	return d.name
}

// IsAvailable checks if the package manager executable is available
func (d *BaseDriver) IsAvailable() bool {
	_, err := exec.LookPath(d.executable)
	return err == nil
}

// RunCommand executes a command and returns the output
func (d *BaseDriver) RunCommand(args ...string) (string, error) {
	cmd := exec.Command(d.executable, args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// RunCommandQuiet executes a command and only returns success/failure
func (d *BaseDriver) RunCommandQuiet(args ...string) error {
	cmd := exec.Command(d.executable, args...)
	return cmd.Run()
}

// CheckCommandSuccess runs a command and returns true if it succeeds
func (d *BaseDriver) CheckCommandSuccess(args ...string) bool {
	return d.RunCommandQuiet(args...) == nil
}

// GetCache returns the package cache
func (d *BaseDriver) GetCache() *PackageCache {
	return d.cache
}

// IsPackageInstalledCached checks if a package is installed using cache when possible
func (d *BaseDriver) IsPackageInstalledCached(packageName string, fetchAllPackages func() (map[string]bool, error)) (bool, error) {
	// Check cache first
	if d.cache.IsValid() {
		if installed, exists := d.cache.GetPackage(packageName); exists {
			return installed, nil
		}
	}

	// Cache miss or invalid - fetch all packages
	packages, err := fetchAllPackages()
	if err != nil {
		return false, err
	}

	// Update cache
	d.cache.SetPackages(packages)

	// Return result for requested package
	installed, exists := packages[packageName]
	return exists && installed, nil
}

// DriverRegistry manages available package drivers
type DriverRegistry struct {
	drivers map[string]PackageDriver
	aliases map[string]string // maps alias names to driver names
}

// NewDriverRegistry creates a new driver registry
func NewDriverRegistry() *DriverRegistry {
	registry := &DriverRegistry{
		drivers: make(map[string]PackageDriver),
		aliases: make(map[string]string),
	}

	// Register all supported drivers
	registry.RegisterDriver(NewChocolateyDriver())
	registry.RegisterDriver(NewScoopDriver())
	registry.RegisterDriver(NewWingetDriver())
	registry.RegisterDriver(NewAptDriver())
	registry.RegisterDriver(NewApkDriver())
	registry.RegisterDriver(NewYumDriver())
	registry.RegisterDriver(NewDnfDriver())
	registry.RegisterDriver(NewBrewDriver())
	registry.RegisterDriver(NewCargoDriver())

	// Register common aliases
	registry.RegisterAlias("choco", "chocolatey")
	registry.RegisterAlias("brew", "homebrew")
	registry.RegisterAlias("rust", "cargo")

	return registry
}

// RegisterDriver registers a package driver
func (r *DriverRegistry) RegisterDriver(driver PackageDriver) {
	r.drivers[driver.Name()] = driver
}

// RegisterAlias registers an alias for a driver name
func (r *DriverRegistry) RegisterAlias(alias, driverName string) {
	r.aliases[alias] = driverName
}

// GetDriver returns a driver by name or alias
func (r *DriverRegistry) GetDriver(name string) (PackageDriver, error) {
	// Try direct name first
	driver, exists := r.drivers[name]
	if exists {
		return driver, nil
	}

	// Try alias
	if driverName, aliasExists := r.aliases[name]; aliasExists {
		driver, exists = r.drivers[driverName]
		if exists {
			return driver, nil
		}
	}

	return nil, fmt.Errorf("driver not found: %s", name)
}

// GetAvailableDrivers returns all available drivers on the current system
func (r *DriverRegistry) GetAvailableDrivers() []PackageDriver {
	var available []PackageDriver

	// Define platform-specific driver order
	var driverOrder []string
	switch runtime.GOOS {
	case "windows":
		driverOrder = []string{
			"winget", "chocolatey", "scoop", // Windows-native managers first
			"cargo",                         // Cross-platform managers
		}
	case "darwin":
		driverOrder = []string{
			"homebrew",                      // macOS-native manager first
			"cargo",                         // Cross-platform managers
		}
	case "linux":
		driverOrder = []string{
			"apt", "apk", "dnf", "yum",     // Linux-native managers first
			"cargo",                         // Cross-platform managers
		}
	default:
		driverOrder = []string{
			"cargo",                         // Cross-platform fallback
		}
	}

	// Add drivers in the platform-specific order
	for _, driverName := range driverOrder {
		if driver, exists := r.drivers[driverName]; exists && driver.IsAvailable() {
			available = append(available, driver)
		}
	}

	// Add any remaining available drivers not in the order list
	// This ensures we don't skip any available drivers
	for _, driver := range r.drivers {
		if driver.IsAvailable() {
			found := false
			for _, existing := range available {
				if existing.Name() == driver.Name() {
					found = true
					break
				}
			}
			if !found {
				available = append(available, driver)
			}
		}
	}

	return available
}

// GetAvailableDriverNames returns names of all available drivers
func (r *DriverRegistry) GetAvailableDriverNames() []string {
	var names []string
	for _, driver := range r.GetAvailableDrivers() {
		names = append(names, driver.Name())
	}
	return names
}

// GetPreferredDriver returns the most preferred available driver from a list
func (r *DriverRegistry) GetPreferredDriver(preferences []string) (PackageDriver, error) {
	available := r.GetAvailableDrivers()
	if len(available) == 0 {
		return nil, fmt.Errorf("no package managers available on this system")
	}

	// Try preferred drivers first (supporting aliases)
	for _, preference := range preferences {
		driver, err := r.GetDriver(preference)
		if err != nil {
			continue
		}
		if driver.IsAvailable() {
			return driver, nil
		}
	}

	// Return the first available driver (now guaranteed to be in consistent order)
	return available[0], nil
}
