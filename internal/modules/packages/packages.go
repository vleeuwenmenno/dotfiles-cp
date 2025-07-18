package packages

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vleeuwenmenno/dotfiles-cp/internal/config"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/logger"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules/packages/drivers"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/platform"
)

// PackagesModule handles package management operations
type PackagesModule struct {
	platformInfo    *platform.PlatformInfo
	driverRegistry  *drivers.DriverRegistry
}

// PackageConfig represents a package configuration
type PackageConfig struct {
	Name            string            `json:"name"`
	State           string            `json:"state"`             // "present" or "absent"
	Managers        map[string]string `json:"managers"`          // package manager specific names
	Prefer          []string          `json:"prefer"`            // preferred package manager order
	Only            []string          `json:"only"`              // only allow these package managers (no fallback)
	CheckSystemWide bool              `json:"check_system_wide"` // check if command is available system-wide before installing

}

// PackageStatus represents the current status of a package
type PackageStatus struct {
	Name          string `json:"name"`
	PackageName   string `json:"package_name"`   // Actual package name used by manager
	Manager       string `json:"manager"`        // Package manager being used
	DesiredState  string `json:"desired_state"`  // "present" or "absent"
	CurrentState  string `json:"current_state"`  // "installed", "not_installed", or "unknown"
	NeedsAction   bool   `json:"needs_action"`   // Whether action is required
	ActionNeeded  string `json:"action_needed"`  // "install", "uninstall", or "none"
}

// New creates a new packages module
func New() *PackagesModule {
	platformInfo, _ := platform.GetPlatformInfo()
	return &PackagesModule{
		platformInfo:   platformInfo,
		driverRegistry: drivers.NewDriverRegistry(),
	}
}

// Name returns the module name
func (m *PackagesModule) Name() string {
	return "packages"
}

// ActionKeys returns the action keys this module handles
func (m *PackagesModule) ActionKeys() []string {
	return []string{"install_package", "uninstall_package", "manage_packages", "add_repo"}
}

// ValidateTask validates a package task configuration
func (m *PackagesModule) ValidateTask(task *config.Task) error {
	switch task.Action {
	case "install_package", "uninstall_package":
		return m.validateSinglePackageTask(task.Config)
	case "manage_packages":
		return m.validateMultiplePackagesTask(task.Config)
	case "add_repo":
		return m.validateAddRepoTask(task.Config)
	default:
		return fmt.Errorf("packages module does not handle action '%s'", task.Action)
	}
}

// ExecuteTask executes a package task
func (m *PackagesModule) ExecuteTask(task *config.Task, ctx *modules.ExecutionContext) error {
	switch task.Action {
	case "install_package":
		return m.executeInstallPackage(task, ctx)
	case "uninstall_package":
		return m.executeUninstallPackage(task, ctx)
	case "manage_packages":
		return m.executeManagePackages(task, ctx)
	case "add_repo":
		return m.executeAddRepo(task, ctx)
	default:
		return fmt.Errorf("packages module does not handle action '%s'", task.Action)
	}
}

// PlanTask returns what the task would do without executing it
func (m *PackagesModule) PlanTask(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	switch task.Action {
	case "install_package":
		return m.planInstallPackage(task, ctx)
	case "uninstall_package":
		return m.planUninstallPackage(task, ctx)
	case "manage_packages":
		return m.planManagePackages(task, ctx)
	case "add_repo":
		return m.planAddRepo(task, ctx)
	default:
		return nil, fmt.Errorf("packages module does not handle action '%s'", task.Action)
	}
}

// validateSinglePackageTask validates configuration for install_package and uninstall_package
func (m *PackagesModule) validateSinglePackageTask(config map[string]interface{}) error {
	if name, exists := config["name"]; !exists || name == "" {
		return fmt.Errorf("package name is required")
	}

	// Validate package manager preferences if specified
	if prefer, exists := config["prefer"]; exists {
		if preferList, ok := prefer.([]interface{}); ok {
			for _, mgr := range preferList {
				if mgrStr, ok := mgr.(string); ok {
					if !m.isValidPackageManager(mgrStr) {
						return fmt.Errorf("invalid package manager: %s", mgrStr)
					}
				}
			}
		}
	}

	// Validate package manager restrictions if specified
	if only, exists := config["only"]; exists {
		if onlyList, ok := only.([]interface{}); ok {
			for _, mgr := range onlyList {
				if mgrStr, ok := mgr.(string); ok {
					if !m.isValidPackageManager(mgrStr) {
						return fmt.Errorf("invalid package manager: %s", mgrStr)
					}
				}
			}
		}
	}

	// Validate that both prefer and only are not specified together
	if _, hasPrefer := config["prefer"]; hasPrefer {
		if _, hasOnly := config["only"]; hasOnly {
			return fmt.Errorf("cannot specify both 'prefer' and 'only' options")
		}
	}

	return nil
}

// validateMultiplePackagesTask validates configuration for manage_packages
func (m *PackagesModule) validateMultiplePackagesTask(config map[string]interface{}) error {
	packages, exists := config["packages"]
	if !exists {
		return fmt.Errorf("packages list is required")
	}

	packagesList, ok := packages.([]interface{})
	if !ok {
		return fmt.Errorf("packages must be a list")
	}

	if len(packagesList) == 0 {
		return fmt.Errorf("packages list cannot be empty")
	}

	for i, pkg := range packagesList {
		pkgConfig, ok := pkg.(map[string]interface{})
		if !ok {
			return fmt.Errorf("package %d must be an object", i)
		}

		if name, exists := pkgConfig["name"]; !exists || name == "" {
			return fmt.Errorf("package %d: name is required", i)
		}

		// Validate state if specified
		if state, exists := pkgConfig["state"]; exists {
			stateStr, ok := state.(string)
			if !ok || (stateStr != "present" && stateStr != "absent") {
				return fmt.Errorf("package %d: state must be 'present' or 'absent'", i)
			}
		}

		// Validate package manager preferences if specified
		if prefer, exists := pkgConfig["prefer"]; exists {
			if preferList, ok := prefer.([]interface{}); ok {
				for _, mgr := range preferList {
					if mgrStr, ok := mgr.(string); ok {
						if !m.isValidPackageManager(mgrStr) {
							return fmt.Errorf("package %d: invalid package manager: %s", i, mgrStr)
						}
					}
				}
			}
		}

		// Validate package manager restrictions if specified
		if only, exists := pkgConfig["only"]; exists {
			if onlyList, ok := only.([]interface{}); ok {
				for _, mgr := range onlyList {
					if mgrStr, ok := mgr.(string); ok {
						if !m.isValidPackageManager(mgrStr) {
							return fmt.Errorf("package %d: invalid package manager: %s", i, mgrStr)
						}
					}
				}
			}
		}

		// Validate that both prefer and only are not specified together
		if _, hasPrefer := pkgConfig["prefer"]; hasPrefer {
			if _, hasOnly := pkgConfig["only"]; hasOnly {
				return fmt.Errorf("package %d: cannot specify both 'prefer' and 'only' options", i)
			}
		}
	}

	return nil
}

// executeInstallPackage installs a single package
func (m *PackagesModule) executeInstallPackage(task *config.Task, ctx *modules.ExecutionContext) error {
	pkg := &PackageConfig{
		Name:  task.Config["name"].(string),
		State: "present",
	}

	if managers, exists := task.Config["managers"]; exists {
		if mgrsMap, ok := managers.(map[string]interface{}); ok {
			pkg.Managers = make(map[string]string)
			for k, v := range mgrsMap {
				pkg.Managers[k] = v.(string)
			}
		}
	}

	if prefer, exists := task.Config["prefer"]; exists {
		if preferList, ok := prefer.([]interface{}); ok {
			pkg.Prefer = make([]string, len(preferList))
			for i, p := range preferList {
				pkg.Prefer[i] = p.(string)
			}
		}
	}

	if only, exists := task.Config["only"]; exists {
		if onlyList, ok := only.([]interface{}); ok {
			pkg.Only = make([]string, len(onlyList))
			for i, o := range onlyList {
				pkg.Only[i] = o.(string)
			}
		}
	}

	// Parse check_system_wide
	if checkSystemWide, exists := task.Config["check_system_wide"]; exists {
		pkg.CheckSystemWide = checkSystemWide.(bool)
	}



	return m.ensurePackageState(pkg, ctx)
}

// executeAddRepo adds a repository/bucket/tap to a package manager
func (m *PackagesModule) executeAddRepo(task *config.Task, ctx *modules.ExecutionContext) error {
	log := logger.Get()

	// Get required repository name
	repoName, exists := task.Config["name"]
	if !exists {
		return fmt.Errorf("name is required for add_repo action")
	}

	repo := repoName.(string)

	// Parse only/prefer to determine which package manager to use
	var driver drivers.PackageDriver
	var err error

	if only, exists := task.Config["only"]; exists {
		if onlyList, ok := only.([]interface{}); ok {
			onlyStrings := make([]string, len(onlyList))
			for i, o := range onlyList {
				onlyStrings[i] = o.(string)
			}
			driver, err = m.driverRegistry.GetOnlyDriver(onlyStrings)
		} else {
			return fmt.Errorf("only must be a list of strings")
		}
	} else if prefer, exists := task.Config["prefer"]; exists {
		if preferList, ok := prefer.([]interface{}); ok {
			preferStrings := make([]string, len(preferList))
			for i, p := range preferList {
				preferStrings[i] = p.(string)
			}
			driver, err = m.driverRegistry.GetPreferredDriver(preferStrings)
		} else {
			return fmt.Errorf("prefer must be a list of strings")
		}
	} else {
		// Use default available driver
		available := m.driverRegistry.GetAvailableDrivers()
		if len(available) == 0 {
			return fmt.Errorf("no package managers available on this system")
		}
		driver = available[0]
	}

	if err != nil {
		return fmt.Errorf("failed to get package manager driver: %w", err)
	}

	log.Debug().
		Str("manager", driver.Name()).
		Str("repo", repo).
		Bool("dry_run", ctx.DryRun).
		Msg("Adding repository")

	if ctx.DryRun {
		fmt.Printf("Would add repository: %s (using %s)\n", repo, driver.Name())
		return nil
	}

	fmt.Printf("Adding repository: %s (using %s)\n", repo, driver.Name())

	// Add the repository
	if err := driver.EnsureRepository(repo); err != nil {
		return fmt.Errorf("failed to add repository %s using %s: %w", repo, driver.Name(), err)
	}

	return nil
}

// validateAddRepoTask validates an add_repo task configuration
func (m *PackagesModule) validateAddRepoTask(config map[string]interface{}) error {
	// Check for required name field
	if _, exists := config["name"]; !exists {
		return fmt.Errorf("name is required for add_repo action")
	}

	// Validate name type
	if _, ok := config["name"].(string); !ok {
		return fmt.Errorf("name must be a string")
	}

	// Validate only field if present
	if only, exists := config["only"]; exists {
		if onlyList, ok := only.([]interface{}); ok {
			for _, manager := range onlyList {
				if managerStr, ok := manager.(string); ok {
					if !m.isValidPackageManager(managerStr) {
						return fmt.Errorf("unsupported package manager in 'only': %s", managerStr)
					}
				} else {
					return fmt.Errorf("all items in 'only' must be strings")
				}
			}
		} else {
			return fmt.Errorf("only must be a list of strings")
		}
	}

	// Validate prefer field if present
	if prefer, exists := config["prefer"]; exists {
		if preferList, ok := prefer.([]interface{}); ok {
			for _, manager := range preferList {
				if managerStr, ok := manager.(string); ok {
					if !m.isValidPackageManager(managerStr) {
						return fmt.Errorf("unsupported package manager in 'prefer': %s", managerStr)
					}
				} else {
					return fmt.Errorf("all items in 'prefer' must be strings")
				}
			}
		} else {
			return fmt.Errorf("prefer must be a list of strings")
		}
	}

	return nil
}

// executeUninstallPackage uninstalls a single package
func (m *PackagesModule) executeUninstallPackage(task *config.Task, ctx *modules.ExecutionContext) error {
	pkg := &PackageConfig{
		Name:  task.Config["name"].(string),
		State: "absent",
	}

	if managers, exists := task.Config["managers"]; exists {
		if mgrsMap, ok := managers.(map[string]interface{}); ok {
			pkg.Managers = make(map[string]string)
			for k, v := range mgrsMap {
				pkg.Managers[k] = v.(string)
			}
		}
	}

	if prefer, exists := task.Config["prefer"]; exists {
		if preferList, ok := prefer.([]interface{}); ok {
			pkg.Prefer = make([]string, len(preferList))
			for i, p := range preferList {
				pkg.Prefer[i] = p.(string)
			}
		}
	}

	if only, exists := task.Config["only"]; exists {
		if onlyList, ok := only.([]interface{}); ok {
			pkg.Only = make([]string, len(onlyList))
			for i, o := range onlyList {
				pkg.Only[i] = o.(string)
			}
		}
	}

	// Parse check_system_wide
	if checkSystemWide, exists := task.Config["check_system_wide"]; exists {
		pkg.CheckSystemWide = checkSystemWide.(bool)
	}

	return m.ensurePackageState(pkg, ctx)
}

// executeManagePackages manages multiple packages
func (m *PackagesModule) executeManagePackages(task *config.Task, ctx *modules.ExecutionContext) error {
	packages := task.Config["packages"].([]interface{})

	for _, pkg := range packages {
		pkgConfig := pkg.(map[string]interface{})

		packageObj := &PackageConfig{
			Name:  pkgConfig["name"].(string),
			State: "present", // default state
		}

		// Parse state
		if state, exists := pkgConfig["state"]; exists {
			packageObj.State = state.(string)
		}

		// Parse managers map
		if managers, exists := pkgConfig["managers"]; exists {
			if mgrsMap, ok := managers.(map[string]interface{}); ok {
				packageObj.Managers = make(map[string]string)
				for k, v := range mgrsMap {
					packageObj.Managers[k] = v.(string)
				}
			}
		}

		// Parse prefer list
		if prefer, exists := pkgConfig["prefer"]; exists {
			if preferList, ok := prefer.([]interface{}); ok {
				packageObj.Prefer = make([]string, len(preferList))
				for i, p := range preferList {
					packageObj.Prefer[i] = p.(string)
				}
			}
		}

		// Parse only list
		if only, exists := pkgConfig["only"]; exists {
			if onlyList, ok := only.([]interface{}); ok {
				packageObj.Only = make([]string, len(onlyList))
				for i, o := range onlyList {
					packageObj.Only[i] = o.(string)
				}
			}
		}

		// Parse check_system_wide
		if checkSystemWide, exists := pkgConfig["check_system_wide"]; exists {
			packageObj.CheckSystemWide = checkSystemWide.(bool)
		}

		if err := m.ensurePackageState(packageObj, ctx); err != nil {
			return fmt.Errorf("failed to manage package %s: %w", packageObj.Name, err)
		}
	}

	return nil
}

// gatherPackageStatus gathers current status information for a package
func (m *PackagesModule) gatherPackageStatus(pkg *PackageConfig) (*PackageStatus, error) {
	log := logger.Get()

	// Check if package is available system-wide first (if enabled)
	if pkg.CheckSystemWide && pkg.State == "present" && !m.isWildcardPattern(pkg.Name) {
		if m.isCommandAvailable(pkg.Name) {
			log.Debug().
				Str("package", pkg.Name).
				Msg("Package found system-wide, skipping package manager check")
			return &PackageStatus{
				Name:         pkg.Name,
				PackageName:  pkg.Name,
				Manager:      "system",
				DesiredState: pkg.State,
				CurrentState: "installed",
				NeedsAction:  false,
				ActionNeeded: "none",
			}, nil
		}
	}

	driver, packageName, err := m.selectPackageDriver(pkg)
	if err != nil {
		return &PackageStatus{
			Name:         pkg.Name,
			PackageName:  pkg.Name,
			Manager:      "none",
			DesiredState: pkg.State,
			CurrentState: "unknown",
			NeedsAction:  false,
			ActionNeeded: "none",
		}, err
	}

	// Handle wildcard patterns
	if m.isWildcardPattern(packageName) {
		return m.gatherWildcardPackageStatus(pkg, driver, packageName)
	}

	isInstalled, err := driver.IsPackageInstalled(packageName)
	if err != nil {
		log.Error().
			Err(err).
			Str("package", pkg.Name).
			Str("driver", driver.Name()).
			Msg("Failed to check if package is installed")
		return &PackageStatus{
			Name:         pkg.Name,
			PackageName:  packageName,
			Manager:      driver.Name(),
			DesiredState: pkg.State,
			CurrentState: "unknown",
			NeedsAction:  false,
			ActionNeeded: "none",
		}, err
	}

	status := &PackageStatus{
		Name:         pkg.Name,
		PackageName:  packageName,
		Manager:      driver.Name(),
		DesiredState: pkg.State,
		NeedsAction:  false,
		ActionNeeded: "none",
	}

	if isInstalled {
		status.CurrentState = "installed"
		if pkg.State == "absent" {
			status.NeedsAction = true
			status.ActionNeeded = "uninstall"
		}
	} else {
		status.CurrentState = "not_installed"
		if pkg.State == "present" {
			status.NeedsAction = true
			status.ActionNeeded = "install"
		}
	}

	log.Debug().
		Str("package", pkg.Name).
		Str("manager", status.Manager).
		Str("current_state", status.CurrentState).
		Str("desired_state", status.DesiredState).
		Bool("needs_action", status.NeedsAction).
		Str("action_needed", status.ActionNeeded).
		Msg("Package status gathered")

	return status, nil
}

// ensurePackageState ensures a package is in the desired state
func (m *PackagesModule) ensurePackageState(pkg *PackageConfig, ctx *modules.ExecutionContext) error {
	log := logger.Get()

	status, err := m.gatherPackageStatus(pkg)
	if err != nil {
		return fmt.Errorf("failed to check package status: %w", err)
	}

	log.Debug().
		Str("package", pkg.Name).
		Str("manager", status.Manager).
		Bool("needs_action", status.NeedsAction).
		Bool("dry_run", ctx.DryRun).
		Msg("Ensuring package state")

	if status.NeedsAction {
		if ctx.DryRun {
			fmt.Printf("Would %s package: %s (using %s)\n", status.ActionNeeded, status.PackageName, status.Manager)
		} else {
			fmt.Printf("%s package: %s (using %s)\n",
				map[string]string{"install": "Installing", "uninstall": "Uninstalling"}[status.ActionNeeded],
				status.PackageName, status.Manager)
		}
		if !ctx.DryRun {
			driver, err := m.driverRegistry.GetDriver(status.Manager)
			if err != nil {
				return fmt.Errorf("failed to get driver for %s: %w", status.Manager, err)
			}

			if status.ActionNeeded == "install" {
				return driver.InstallPackage(status.PackageName)
			} else if status.ActionNeeded == "uninstall" {
				// Handle wildcard patterns for uninstall
				if m.isWildcardPattern(status.PackageName) {
					return m.uninstallWildcardPackages(driver, status.PackageName, ctx)
				}
				return driver.UninstallPackage(status.PackageName)
			}
		}
	} else {
		if status.DesiredState == "present" {
			fmt.Printf("Package already installed: %s via %s (skipping)\n", status.PackageName, status.Manager)
		} else {
			fmt.Printf("Package already absent: %s (skipping)\n", status.PackageName)
		}
	}

	return nil
}

// selectPackageDriver selects the best package driver for a package
func (m *PackagesModule) selectPackageDriver(pkg *PackageConfig) (drivers.PackageDriver, string, error) {
	log := logger.Get()

	if m.driverRegistry == nil {
		return nil, "", fmt.Errorf("driver registry is not initialized")
	}

	var driver drivers.PackageDriver
	var err error

	// Use only constraint if specified, otherwise use prefer
	if len(pkg.Only) > 0 {
		driver, err = m.driverRegistry.GetOnlyDriver(pkg.Only)
		if err != nil {
			return nil, "", fmt.Errorf("failed to find required package manager for %s: %w", pkg.Name, err)
		}
	} else {
		driver, err = m.driverRegistry.GetPreferredDriver(pkg.Prefer)
		if err != nil {
			return nil, "", err
		}
	}

	if driver == nil {
		return nil, "", fmt.Errorf("no suitable package manager found for %s", pkg.Name)
	}

	packageName := m.getPackageNameForManager(pkg, driver.Name())

	// Log driver selection for debugging
	log.Debug().
		Str("package", pkg.Name).
		Str("selected_driver", driver.Name()).
		Str("package_name", packageName).
		Interface("preferences", pkg.Prefer).
		Interface("only", pkg.Only).
		Msg("Selected package driver")

	return driver, packageName, nil
}

// getPackageNameForManager gets the package name for a specific manager
func (m *PackagesModule) getPackageNameForManager(pkg *PackageConfig, manager string) string {
	if pkg.Managers != nil {
		if name, exists := pkg.Managers[manager]; exists {
			return name
		}
	}
	return pkg.Name // fallback to generic name
}



// isValidPackageManager checks if a package manager name is valid
func (m *PackagesModule) isValidPackageManager(manager string) bool {
	validManagers := []string{
		"winget", "chocolatey", "scoop",    // Windows
		"homebrew",                         // macOS
		"apt", "apk", "yum", "dnf",        // Linux
		"cargo",                           // Cross-platform (Rust)
	}

	for _, valid := range validManagers {
		if manager == valid {
			return true
		}
	}
	return false
}

// Planning functions for dry-run support
func (m *PackagesModule) planInstallPackage(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	pkg := &PackageConfig{
		Name:  task.Config["name"].(string),
		State: "present",
	}

	if managers, exists := task.Config["managers"]; exists {
		if mgrsMap, ok := managers.(map[string]interface{}); ok {
			pkg.Managers = make(map[string]string)
			for k, v := range mgrsMap {
				pkg.Managers[k] = v.(string)
			}
		}
	}

	if prefer, exists := task.Config["prefer"]; exists {
		if preferList, ok := prefer.([]interface{}); ok {
			pkg.Prefer = make([]string, len(preferList))
			for i, p := range preferList {
				pkg.Prefer[i] = p.(string)
			}
		}
	}

	if only, exists := task.Config["only"]; exists {
		if onlyList, ok := only.([]interface{}); ok {
			pkg.Only = make([]string, len(onlyList))
			for i, o := range onlyList {
				pkg.Only[i] = o.(string)
			}
		}
	}

	// Parse check_system_wide
	if checkSystemWide, exists := task.Config["check_system_wide"]; exists {
		pkg.CheckSystemWide = checkSystemWide.(bool)
	}

	return m.planPackageChange(pkg, ctx)
}

func (m *PackagesModule) planUninstallPackage(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	pkg := &PackageConfig{
		Name:  task.Config["name"].(string),
		State: "absent",
	}

	if managers, exists := task.Config["managers"]; exists {
		if mgrsMap, ok := managers.(map[string]interface{}); ok {
			pkg.Managers = make(map[string]string)
			for k, v := range mgrsMap {
				pkg.Managers[k] = v.(string)
			}
		}
	}

	if prefer, exists := task.Config["prefer"]; exists {
		if preferList, ok := prefer.([]interface{}); ok {
			pkg.Prefer = make([]string, len(preferList))
			for i, p := range preferList {
				pkg.Prefer[i] = p.(string)
			}
		}
	}

	if only, exists := task.Config["only"]; exists {
		if onlyList, ok := only.([]interface{}); ok {
			pkg.Only = make([]string, len(onlyList))
			for i, o := range onlyList {
				pkg.Only[i] = o.(string)
			}
		}
	}

	// Parse check_system_wide
	if checkSystemWide, exists := task.Config["check_system_wide"]; exists {
		pkg.CheckSystemWide = checkSystemWide.(bool)
	}

	return m.planPackageChange(pkg, ctx)
}

func (m *PackagesModule) planManagePackages(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	packages := task.Config["packages"].([]interface{})

	plan := &modules.TaskPlan{
		TaskID:      task.ID,
		Action:      task.Action,
		Description: fmt.Sprintf("Manage %d packages", len(packages)),
		Changes:     []string{},
		WillSkip:    false,
	}

	actionablePackages := 0
	skippedPackages := 0

	for _, pkg := range packages {
		pkgConfig := pkg.(map[string]interface{})

		packageObj := &PackageConfig{
			Name:  pkgConfig["name"].(string),
			State: "present", // default state
		}

		// Parse state
		if state, exists := pkgConfig["state"]; exists {
			packageObj.State = state.(string)
		}

		// Parse managers map
		if managers, exists := pkgConfig["managers"]; exists {
			if mgrsMap, ok := managers.(map[string]interface{}); ok {
				packageObj.Managers = make(map[string]string)
				for k, v := range mgrsMap {
					packageObj.Managers[k] = v.(string)
				}
			}
		}

		// Parse prefer list
		if prefer, exists := pkgConfig["prefer"]; exists {
			if preferList, ok := prefer.([]interface{}); ok {
				packageObj.Prefer = make([]string, len(preferList))
				for i, p := range preferList {
					packageObj.Prefer[i] = p.(string)
				}
			}
		}

		// Parse only list
		if only, exists := pkgConfig["only"]; exists {
			if onlyList, ok := only.([]interface{}); ok {
				packageObj.Only = make([]string, len(onlyList))
				for i, o := range onlyList {
					packageObj.Only[i] = o.(string)
				}
			}
		}

		// Parse check_system_wide
		if checkSystemWide, exists := pkgConfig["check_system_wide"]; exists {
			packageObj.CheckSystemWide = checkSystemWide.(bool)
		}

		pkgPlan, err := m.planPackageChange(packageObj, ctx)
		if err != nil {
			plan.WillSkip = true
			plan.SkipReason = fmt.Sprintf("Failed to plan package %s: %v", packageObj.Name, err)
			return plan, nil
		}

		if !pkgPlan.WillSkip {
			plan.Changes = append(plan.Changes, pkgPlan.Changes...)
			actionablePackages++
		} else {
			skippedPackages++
			// Collect skip reasons for individual packages
			if plan.SkipReason == "" {
				plan.SkipReason = pkgPlan.SkipReason
			} else {
				plan.SkipReason += "; " + pkgPlan.SkipReason
			}
		}
	}

	if len(plan.Changes) == 0 {
		plan.WillSkip = true
		if skippedPackages > 0 && plan.SkipReason == "" {
			plan.SkipReason = fmt.Sprintf("All %d packages already in desired state", skippedPackages)
		} else if plan.SkipReason == "" {
			plan.SkipReason = "No packages to process"
		}
	} else {
		// Update description to show actionable vs skipped counts
		if skippedPackages > 0 {
			plan.Description = fmt.Sprintf("Manage %d packages (%d changes, %d already correct)",
				len(packages), actionablePackages, skippedPackages)
		} else {
			plan.Description = fmt.Sprintf("Manage %d packages", actionablePackages)
		}
	}

	return plan, nil
}

func (m *PackagesModule) planPackageChange(pkg *PackageConfig, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	plan := &modules.TaskPlan{
		TaskID:      fmt.Sprintf("package-%s", pkg.Name),
		Action:      fmt.Sprintf("ensure_%s", pkg.State),
		Description: fmt.Sprintf("Ensure package %s is %s", pkg.Name, pkg.State),
		Changes:     []string{},
		WillSkip:    false,
	}

	status, err := m.gatherPackageStatus(pkg)
	if err != nil {
		return nil, fmt.Errorf("cannot determine package status: %w", err)
	}

	if status.NeedsAction {
		actionVerb := map[string]string{
			"install":   "Install",
			"uninstall": "Uninstall",
		}[status.ActionNeeded]
		plan.Changes = append(plan.Changes, fmt.Sprintf("%s package %s using %s", actionVerb, status.PackageName, status.Manager))
	} else {
		plan.WillSkip = true
		if status.DesiredState == "present" {
			plan.SkipReason = fmt.Sprintf("Package %s already installed via %s", status.PackageName, status.Manager)
		} else {
			plan.SkipReason = fmt.Sprintf("Package %s already absent", status.PackageName)
		}
	}

	return plan, nil
}

// ExplainAction returns documentation for a specific action
func (m *PackagesModule) ExplainAction(action string) (*modules.ActionDocumentation, error) {
	switch action {
	case "install_package":
		return &modules.ActionDocumentation{
			Action:      "install_package",
			Description: "Install a single package using the system's package manager",
			Parameters: []modules.ActionParameter{
				{
					Name:        "name",
					Type:        "string",
					Required:    true,
					Description: "Name of the package to install",
				},
				{
					Name:        "managers",
					Type:        "map[string]string",
					Required:    false,
					Description: "Package manager specific names (e.g., {\"winget\": \"Git.Git\", \"brew\": \"git\"})",
				},
				{
					Name:        "prefer",
					Type:        "[]string",
					Required:    false,
					Description: "Preferred package manager order (e.g., [\"winget\", \"brew\"])",
				},
				{
					Name:        "only",
					Type:        "[]string",
					Required:    false,
					Description: "Only allow these package managers, no fallbacks (e.g., [\"cargo\", \"apt\"])",
				},
			},
			Examples: []modules.ActionExample{
				{
					Description: "Install git package",
					Config: map[string]interface{}{
						"name": "git",
					},
				},
				{
					Description: "Install Node.js with manager-specific names",
					Config: map[string]interface{}{
						"name": "nodejs",
						"managers": map[string]string{
							"winget": "OpenJS.NodeJS",
							"brew":   "node",
							"apt":    "nodejs",
						},
					},
				},
				{
					Description: "Install starship only through cargo or apt",
					Config: map[string]interface{}{
						"name": "starship",
						"only": []string{"cargo", "apt"},
					},
				},
			},
		}, nil
	case "uninstall_package":
		return &modules.ActionDocumentation{
			Action:      "uninstall_package",
			Description: "Uninstall a single package using the system's package manager",
			Parameters: []modules.ActionParameter{
				{
					Name:        "name",
					Type:        "string",
					Required:    true,
					Description: "Name of the package to uninstall",
				},
				{
					Name:        "managers",
					Type:        "map[string]string",
					Required:    false,
					Description: "Package manager specific names",
				},
				{
					Name:        "prefer",
					Type:        "[]string",
					Required:    false,
					Description: "Preferred package manager order",
				},
				{
					Name:        "only",
					Type:        "[]string",
					Required:    false,
					Description: "Only allow these package managers, no fallbacks",
				},
			},
			Examples: []modules.ActionExample{
				{
					Description: "Uninstall git package",
					Config: map[string]interface{}{
						"name": "git",
					},
				},
			},
		}, nil
	case "manage_packages":
		return &modules.ActionDocumentation{
			Action:      "manage_packages",
			Description: "Manage multiple packages with different states (install/uninstall)",
			Parameters: []modules.ActionParameter{
				{
					Name:        "packages",
					Type:        "[]object",
					Required:    true,
					Description: "List of package configurations",
				},
			},
			Examples: []modules.ActionExample{
				{
					Description: "Manage multiple packages",
					Config: map[string]interface{}{
						"packages": []map[string]interface{}{
							{
								"name":  "git",
								"state": "present",
							},
							{
								"name":  "curl",
								"state": "present",
							},
							{
								"name":  "old-package",
								"state": "absent",
							},
						},
					},
				},
			},
		}, nil
	case "add_repo":
		return &modules.ActionDocumentation{
			Action:      "add_repo",
			Description: "Add a repository, bucket, or tap to a package manager",
			Parameters: []modules.ActionParameter{
				{
					Name:        "name",
					Type:        "string",
					Required:    true,
					Description: "Name of the repository/bucket/tap to add",
				},
				{
					Name:        "only",
					Type:        "[]string",
					Required:    false,
					Description: "Only use these package managers (no fallback)",
				},
				{
					Name:        "prefer",
					Type:        "[]string",
					Required:    false,
					Description: "Preferred package managers in order of preference",
				},
			},
			Examples: []modules.ActionExample{
				{
					Description: "Add Scoop extras bucket",
					Config: map[string]interface{}{
						"name": "extras",
						"only": []string{"scoop"},
					},
				},
				{
					Description: "Add Homebrew tap",
					Config: map[string]interface{}{
						"name": "homebrew/cask-fonts",
						"prefer": []string{"homebrew"},
					},
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// ListActions returns documentation for all actions supported by this module
func (m *PackagesModule) ListActions() []*modules.ActionDocumentation {
	actions := []string{"install_package", "uninstall_package", "manage_packages", "add_repo"}
	docs := make([]*modules.ActionDocumentation, len(actions))

	for i, action := range actions {
		doc, _ := m.ExplainAction(action)
		docs[i] = doc
	}

	return docs
}

// isCommandAvailable checks if a command is available system-wide in PATH
func (m *PackagesModule) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// isWildcardPattern checks if a package name contains wildcard characters
func (m *PackagesModule) isWildcardPattern(name string) bool {
	return strings.ContainsAny(name, "*?")
}

// gatherWildcardPackageStatus handles package status for wildcard patterns
func (m *PackagesModule) gatherWildcardPackageStatus(pkg *PackageConfig, driver drivers.PackageDriver, pattern string) (*PackageStatus, error) {
	// Get all installed packages using the new interface method
	allPackages, err := driver.GetAllInstalledPackages()
	if err != nil {
		return &PackageStatus{
			Name:         pkg.Name,
			PackageName:  pattern,
			Manager:      driver.Name(),
			DesiredState: pkg.State,
			CurrentState: "unknown",
			NeedsAction:  false,
			ActionNeeded: "none",
		}, fmt.Errorf("failed to fetch installed packages for wildcard matching: %w", err)
	}

	// Find matching packages
	var matchingPackages []string
	for packageName := range allPackages {
		matched, err := filepath.Match(pattern, packageName)
		if err != nil {
			continue // Skip invalid patterns
		}
		if matched {
			matchingPackages = append(matchingPackages, packageName)
		}
	}

	status := &PackageStatus{
		Name:         pkg.Name,
		PackageName:  pattern,
		Manager:      driver.Name(),
		DesiredState: pkg.State,
		NeedsAction:  false,
		ActionNeeded: "none",
	}

	// Determine status based on matches and desired state
	hasMatches := len(matchingPackages) > 0

	if pkg.State == "present" {
		if hasMatches {
			status.CurrentState = "installed"
			// For wildcard install: if any packages match, consider it satisfied
		} else {
			status.CurrentState = "not_installed"
			status.NeedsAction = true
			status.ActionNeeded = "install"
		}
	} else if pkg.State == "absent" {
		if hasMatches {
			status.CurrentState = "installed"
			status.NeedsAction = true
			status.ActionNeeded = "uninstall"
		} else {
			status.CurrentState = "not_installed"
			// Already in desired state - no action needed
		}
	}

	return status, nil
}

// uninstallWildcardPackages handles uninstalling packages that match a wildcard pattern
func (m *PackagesModule) uninstallWildcardPackages(driver drivers.PackageDriver, pattern string, ctx *modules.ExecutionContext) error {
	// Get all installed packages
	allPackages, err := driver.GetAllInstalledPackages()
	if err != nil {
		return fmt.Errorf("failed to get installed packages for wildcard uninstall: %w", err)
	}

	// Find matching packages
	var matchingPackages []string
	for packageName := range allPackages {
		matched, err := filepath.Match(pattern, packageName)
		if err != nil {
			continue // Skip invalid patterns
		}
		if matched {
			matchingPackages = append(matchingPackages, packageName)
		}
	}

	if len(matchingPackages) == 0 {
		fmt.Printf("No packages found matching pattern: %s\n", pattern)
		return nil
	}

	// Uninstall each matching package
	for _, pkgName := range matchingPackages {
		fmt.Printf("Uninstalling matched package: %s (using %s)\n", pkgName, driver.Name())
		err := driver.UninstallPackage(pkgName)
		if err != nil {
			return fmt.Errorf("failed to uninstall package %s: %w", pkgName, err)
		}
	}

	return nil
}

// planAddRepo returns what adding a repository would do without executing it
func (m *PackagesModule) planAddRepo(task *config.Task, ctx *modules.ExecutionContext) (*modules.TaskPlan, error) {
	repoName := task.Config["name"].(string)

	plan := &modules.TaskPlan{
		TaskID:      task.ID,
		Action:      task.Action,
		Description: fmt.Sprintf("Add repository: %s", repoName),
		Changes:     []string{},
		WillSkip:    false,
	}

	// Determine which package manager will be used
	var driver drivers.PackageDriver
	var err error

	if only, exists := task.Config["only"]; exists {
		if onlyList, ok := only.([]interface{}); ok {
			onlyStrings := make([]string, len(onlyList))
			for i, o := range onlyList {
				onlyStrings[i] = o.(string)
			}
			driver, err = m.driverRegistry.GetOnlyDriver(onlyStrings)
		}
	} else if prefer, exists := task.Config["prefer"]; exists {
		if preferList, ok := prefer.([]interface{}); ok {
			preferStrings := make([]string, len(preferList))
			for i, p := range preferList {
				preferStrings[i] = p.(string)
			}
			driver, err = m.driverRegistry.GetPreferredDriver(preferStrings)
		}
	} else {
		available := m.driverRegistry.GetAvailableDrivers()
		if len(available) > 0 {
			driver = available[0]
		}
	}

	if err != nil || driver == nil {
		plan.WillSkip = true
		plan.SkipReason = "No suitable package manager available"
		return plan, nil
	}

	// Check if repository is already available
	isAvailable, err := driver.IsRepositoryAvailable(repoName)
	if err != nil {
		// If we can't check availability, assume we need to add it
		plan.Changes = append(plan.Changes, fmt.Sprintf("Add repository %s using %s (unable to verify current state)", repoName, driver.Name()))
		return plan, nil
	}

	if isAvailable {
		// Repository is already available, no action needed
		plan.WillSkip = true
		plan.SkipReason = fmt.Sprintf("Repository %s already available in %s", repoName, driver.Name())
		return plan, nil
	}

	plan.Changes = append(plan.Changes, fmt.Sprintf("Add repository %s using %s", repoName, driver.Name()))

	return plan, nil
}
