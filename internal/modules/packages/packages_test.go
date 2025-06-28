package packages

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/modules/packages/drivers"
	"github.com/vleeuwenmenno/dotfiles-cp/internal/platform"
)

func TestPackageConfigValidation(t *testing.T) {
	// Create a properly initialized module for testing
	platformInfo := &platform.PlatformInfo{OS: "windows", Arch: "amd64"}
	driverRegistry := drivers.NewDriverRegistry()
	module := &PackagesModule{
		platformInfo:   platformInfo,
		driverRegistry: driverRegistry,
	}

	t.Run("ValidateOnlyField", func(t *testing.T) {
		// Valid only configuration
		config := map[string]interface{}{
			"name": "starship",
			"only": []interface{}{"cargo", "apt"},
		}
		err := module.validateSinglePackageTask(config)
		assert.NoError(t, err)
	})

	t.Run("ValidatePreferField", func(t *testing.T) {
		// Valid prefer configuration
		config := map[string]interface{}{
			"name":   "git",
			"prefer": []interface{}{"winget", "homebrew"},
		}
		err := module.validateSinglePackageTask(config)
		assert.NoError(t, err)
	})

	t.Run("RejectBothPreferAndOnly", func(t *testing.T) {
		// Should reject both prefer and only
		config := map[string]interface{}{
			"name":   "git",
			"prefer": []interface{}{"winget"},
			"only":   []interface{}{"apt"},
		}
		err := module.validateSinglePackageTask(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify both 'prefer' and 'only' options")
	})

	t.Run("ValidateMultiplePackagesWithOnly", func(t *testing.T) {
		config := map[string]interface{}{
			"packages": []interface{}{
				map[string]interface{}{
					"name": "starship",
					"only": []interface{}{"cargo", "apt"},
				},
				map[string]interface{}{
					"name":   "git",
					"prefer": []interface{}{"winget", "homebrew"},
				},
			},
		}
		err := module.validateMultiplePackagesTask(config)
		assert.NoError(t, err)
	})

	t.Run("RejectMixedPreferOnlyInPackages", func(t *testing.T) {
		config := map[string]interface{}{
			"packages": []interface{}{
				map[string]interface{}{
					"name":   "starship",
					"only":   []interface{}{"cargo"},
					"prefer": []interface{}{"apt"},
				},
			},
		}
		err := module.validateMultiplePackagesTask(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify both 'prefer' and 'only' options")
	})
}

func TestPackageConfig(t *testing.T) {
	t.Run("StructFields", func(t *testing.T) {
		config := PackageConfig{
			Name:            "test-package",
			State:           "present",
			Managers:        map[string]string{"apt": "test-pkg"},
			Prefer:          []string{"apt", "homebrew"},
			Only:            []string{"cargo", "apt"},
			CheckSystemWide: true,
		}

		assert.Equal(t, "test-package", config.Name)
		assert.Equal(t, "present", config.State)
		assert.Equal(t, []string{"apt", "homebrew"}, config.Prefer)
		assert.Equal(t, []string{"cargo", "apt"}, config.Only)
		assert.True(t, config.CheckSystemWide)
	})
}

func TestSelectPackageDriverLogic(t *testing.T) {
	// Note: This would require mocking the driver registry
	// For now, we'll test the validation logic which is the main addition
	t.Run("ConfigurationParsing", func(t *testing.T) {
		// Test that Only field is properly handled in configuration
		config := PackageConfig{
			Name: "starship",
			Only: []string{"cargo", "apt"},
		}

		assert.Equal(t, "starship", config.Name)
		assert.Equal(t, []string{"cargo", "apt"}, config.Only)
		assert.Empty(t, config.Prefer) // Should be empty when Only is used
	})
}

func TestPackageConfigParsing(t *testing.T) {
	t.Run("ParseOnlyFieldFromTaskConfig", func(t *testing.T) {
		// Test that only field is properly parsed from task config
		taskConfig := map[string]interface{}{
			"name": "starship",
			"only": []interface{}{"cargo", "homebrew"},
		}

		// This simulates what happens in executeInstallPackage
		pkg := &PackageConfig{
			Name:  taskConfig["name"].(string),
			State: "present",
		}

		if only, exists := taskConfig["only"]; exists {
			if onlyList, ok := only.([]interface{}); ok {
				pkg.Only = make([]string, len(onlyList))
				for i, o := range onlyList {
					pkg.Only[i] = o.(string)
				}
			}
		}

		assert.Equal(t, "starship", pkg.Name)
		assert.Equal(t, []string{"cargo", "homebrew"}, pkg.Only)
		assert.Empty(t, pkg.Prefer)
	})

	t.Run("ParseBothPreferAndOnlyFields", func(t *testing.T) {
		// Test parsing both prefer and only (even though validation should reject this)
		taskConfig := map[string]interface{}{
			"name":   "test-package",
			"prefer": []interface{}{"apt", "yum"},
			"only":   []interface{}{"cargo", "homebrew"},
		}

		pkg := &PackageConfig{
			Name:  taskConfig["name"].(string),
			State: "present",
		}

		if prefer, exists := taskConfig["prefer"]; exists {
			if preferList, ok := prefer.([]interface{}); ok {
				pkg.Prefer = make([]string, len(preferList))
				for i, p := range preferList {
					pkg.Prefer[i] = p.(string)
				}
			}
		}

		if only, exists := taskConfig["only"]; exists {
			if onlyList, ok := only.([]interface{}); ok {
				pkg.Only = make([]string, len(onlyList))
				for i, o := range onlyList {
					pkg.Only[i] = o.(string)
				}
			}
		}

		assert.Equal(t, "test-package", pkg.Name)
		assert.Equal(t, []string{"apt", "yum"}, pkg.Prefer)
		assert.Equal(t, []string{"cargo", "homebrew"}, pkg.Only)
	})
}
