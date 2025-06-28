# Debugging & Troubleshooting Guide

This guide helps you debug and troubleshoot common issues with the dotfiles manager, focusing on variables, templates, and configuration problems.

## 📖 **Table of Contents**

- [Quick Debugging Commands](#quick-debugging-commands)
- [Variable Debugging](#variable-debugging)
- [Template Debugging](#template-debugging)
- [Configuration Issues](#configuration-issues)
- [Import System Debugging](#import-system-debugging)
- [Platform Detection Issues](#platform-detection-issues)
- [Common Error Messages](#common-error-messages)
- [Advanced Debugging Techniques](#advanced-debugging-techniques)
- [Performance Issues](#performance-issues)
- [Getting Help](#getting-help)

## 🚀 **Quick Debugging Commands**

### **Essential Debug Commands**

```bash
# Check platform detection
dotfiles info

# Validate configuration
dotfiles validate

# Show all variables and their sources
dotfiles variables sources

# List all loaded variables
dotfiles variables list

# Trace specific variable
dotfiles variables trace user.name

# Test with verbose logging
dotfiles variables list --verbose
```

### **Emergency Diagnostics**

```bash
# Quick health check
dotfiles info && dotfiles validate

# Show variable loading process
dotfiles variables sources --verbose

# Test variable resolution
dotfiles variables get user.name --verbose
```

## 🔍 **Variable Debugging**

### **Variable Not Found**

**Problem**: `dotfiles variables get user.name` returns "Variable not found"

**Debug Steps**:

```bash
# 1. Check if variable exists in any form
dotfiles variables list | grep -i user

# 2. Check all loaded sources
dotfiles variables sources

# 3. Trace the variable path
dotfiles variables trace user

# 4. Check raw variable files
cat variables/global.yaml | grep -A5 user
```

**Common Causes**:
- Typo in variable name
- Variable in unloaded file
- YAML syntax error preventing file load
- Missing import in index.yaml

### **Variable Has Wrong Value**

**Problem**: Variable shows unexpected value

**Debug Steps**:

```bash
# 1. Trace variable precedence (shows rendered values)
dotfiles variables trace user.email

# 2. Check raw template syntax for debugging
dotfiles variables trace user.email --raw

# 3. Check all sources that define this variable
dotfiles variables sources | grep -A10 "user"

# 4. Test with different environments
dotfiles variables get user.email --env DOTFILES_ENV=work
dotfiles variables get user.email --env DOTFILES_ENV=""

# 5. Check template processing
dotfiles variables get user.email --format json
```

**Example Debug Session**:
```bash
$ dotfiles variables trace user.email
Variable: user.email
Sources (rendered values):

1. Source: C:\Users\menno\.dotfiles\variables\global.yaml
   Value: user@example.com

2. Source: C:\Users\menno\.dotfiles\variables\environments\work.yaml
   Value: menno@company.com

Final value: menno@company.com (from work.yaml)

$ dotfiles variables trace user.email --raw
Variable: user.email
Sources (raw template values):

1. Source: C:\Users\menno\.dotfiles\variables\global.yaml
   Raw Value: user@example.com

2. Source: C:\Users\menno\.dotfiles\variables\environments\work.yaml
   Raw Value: {{ .user.name }}@company.com

Final raw value: {{ .user.name }}@company.com (from work.yaml)

Final processed result: menno@company.com
```

### **Template Processing Issues**

**Problem**: Variables show raw template syntax like `{{ .User.Home }}`

**Debug Steps**:

```bash
# 1. Check if template processing is working
dotfiles variables get paths.home

# 2. Compare raw vs processed values
dotfiles variables trace paths.home --raw
dotfiles variables trace paths.home

# 3. Test with verbose output
dotfiles variables get paths.home --verbose

# 4. Check for template syntax errors
dotfiles validate

# 5. Test environment variable access
echo $USERPROFILE  # Windows
echo $HOME         # Linux/macOS
```

**Common Template Errors**:

```yaml
# Wrong: Invalid template syntax
path: "{{ .User.Home }"  # Missing closing brace

# Wrong: Accessing non-existent variable
path: "{{ .NonExistent.Var }}"

# Right: Valid template with fallback
path: "{{ .User.Home | default \"/home/user\" }}"
```

## 🎨 **Template Debugging**

### **Template Syntax Errors**

**Problem**: Templates fail to process

**Debug Commands**:

```bash
# Check template validation
dotfiles validate

# Test variable resolution step by step
dotfiles variables get directories.projects
dotfiles variables trace directories.projects

# Check raw template for syntax issues
dotfiles variables trace directories.projects --raw
```

**Common Syntax Issues**:

```yaml
# Wrong: Unmatched braces
value: "{{ .User.Home }"

# Wrong: Invalid function call
value: "{{ pathJoin(.User.Home, "Projects") }}"

# Right: Correct syntax
value: "{{ pathJoin .User.Home \"Projects\" }}"
```

### **Path Issues**

**Problem**: Paths have wrong slashes or don't resolve correctly

**Debug Steps**:

```bash
# 1. Check path variables
dotfiles variables get paths.home
dotfiles variables get directories.projects

# 2. Test pathJoin function - compare raw vs processed
dotfiles variables trace directories.projects --raw
dotfiles variables trace directories.projects

# 3. Test pathJoin function with verbose output
dotfiles variables get directories.projects --verbose

# 4. Check platform detection
dotfiles info
```

**Path Debugging Examples**:

```bash
# Check if pathJoin is working correctly
$ dotfiles variables get directories.projects
directories.projects: C:\Users\menno\Projects

# Check raw template before processing
$ dotfiles variables trace directories.projects --raw
Variable: directories.projects
Sources (raw template values):

1. Source: C:\Users\menno\.dotfiles\variables\global.yaml
   Raw Value: {{ pathJoin .User.Home "Projects" }}

Final processed result: C:\Users\menno\Projects

# Check processed value
$ dotfiles variables trace directories.projects
Variable: directories.projects
Sources (rendered values):

1. Source: C:\Users\menno\.dotfiles\variables\global.yaml
   Value: C:\Users\menno\Projects
```

## ⚙️ **Configuration Issues**

### **Configuration File Not Found**

**Problem**: `dotfiles` commands fail with "config file not found"

**Debug Steps**:

```bash
# 1. Check current directory
pwd

# 2. Look for config files
ls -la dotfiles.yaml
ls -la .dotfiles.yaml

# 3. Check search paths
dotfiles info  # Will show search locations
```

**Config Search Order**:
1. `./dotfiles.yaml`
2. `./dotfiles.yml`
3. `./.dotfiles.yaml`
4. `./.dotfiles.yml`
5. `~/.dotfiles/dotfiles.yaml`
6. `~/.config/dotfiles/dotfiles.yaml`

### **Invalid Configuration**

**Problem**: Configuration validation fails

**Debug Steps**:

```bash
# 1. Validate configuration
dotfiles validate

# 2. Check YAML syntax
# Use online YAML validator or:
python -c "import yaml; yaml.safe_load(open('dotfiles.yaml'))"

# 3. Check required fields
cat dotfiles.yaml
```

**Common Config Issues**:

```yaml
# Wrong: Missing required fields
metadata:
  name: "My Dotfiles"
# Missing paths and settings sections

# Right: Complete minimal config
metadata:
  name: "My Dotfiles"
  version: "1.0.0"
  author: "User"
paths:
  variables_dir: "variables"
  variables_index: "index.yaml"
  templates_dir: "templates"
  templates_index: "index.yaml"
settings:
  log_level: "info"
  dry_run: false
```

## 📤 **Import System Debugging**

### **Circular Import Detection**

**Problem**: "circular import detected" error

**Debug Steps**:

```bash
# The error will show the import chain
# Example error:
# circular import detected: index.yaml -> global.yaml -> work.yaml -> index.yaml

# 1. Check import chain in error message
# 2. Review import statements in each file
# 3. Remove circular reference
```

**Example Circular Import**:

```yaml
# variables/index.yaml
imports:
  - path: "global.yaml"

# variables/global.yaml  
imports:
  - path: "work.yaml"

# variables/work.yaml
imports:
  - path: "index.yaml"  # CIRCULAR!
```

### **Conditional Import Issues**

**Problem**: Expected file not loading

**Debug Steps**:

```bash
# 1. Test condition manually
dotfiles variables list --env DOTFILES_ENV=work

# 2. Check condition syntax
# 3. Verify environment variables
echo $DOTFILES_ENV

# 4. Test with verbose logging
dotfiles variables list --verbose
```

**Condition Testing**:

```yaml
# Test different condition patterns
imports:
  # Simple existence check
  - path: "work.yaml"
    condition: "ne .Env.DOTFILES_ENV \"\""
  
  # Specific value check
  - path: "work.yaml"
    condition: "eq .Env.DOTFILES_ENV \"work\""
  
  # Complex condition
  - path: "gaming.yaml"
    condition: "and (eq .Platform.OS \"windows\") (ne .Env.GAMING_PC \"\")"
```

## 🖥️ **Platform Detection Issues**

### **Wrong Platform Detection**

**Problem**: Variables loading for wrong platform

**Debug Steps**:

```bash
# 1. Check detected platform
dotfiles info

# 2. Override platform for testing
dotfiles variables list --platform windows
dotfiles variables list --platform linux
dotfiles variables list --platform darwin

# 3. Check platform-specific files
ls variables/platforms/

# 4. Trace platform-specific variables
dotfiles variables trace shell.type
dotfiles variables trace shell.type --raw
```

**Platform Override Testing**:

```bash
# Test different platform combinations
dotfiles variables list --platform windows --shell powershell
dotfiles variables list --platform linux --shell bash
dotfiles variables list --platform darwin --shell zsh
```

## ❌ **Common Error Messages**

### **Variable Errors**

| Error | Cause | Solution |
|-------|-------|----------|
| `Variable not found: user.name` | Variable doesn't exist | Check spelling, verify file loaded |
| `failed to process template` | Template syntax error | Check Go template syntax |
| `circular import detected` | Import loop | Remove circular references |
| `variable file does not exist` | Missing file | Check file path and spelling |

### **Configuration Errors**

| Error | Cause | Solution |
|-------|-------|----------|
| `config file not found` | No dotfiles.yaml in search paths | Create config or run from correct directory |
| `paths section is required` | Missing paths in config | Add paths section to dotfiles.yaml |
| `failed to unmarshal config` | Invalid YAML syntax | Check YAML formatting |

### **Template Errors**

| Error | Cause | Solution |
|-------|-------|----------|
| `failed to parse template` | Invalid Go template syntax | Check template syntax |
| `no such key` | Accessing non-existent variable | Check variable exists |
| `wrong type for value` | Type mismatch in template | Check variable types |

## 🔬 **Advanced Debugging Techniques**

### **Verbose Logging**

```bash
# Enable verbose logging for detailed output
dotfiles variables list --verbose
dotfiles apply --verbose
dotfiles validate --verbose
```

### **Manual Variable Testing**

```bash
# Test specific variable combinations
dotfiles variables get user.name --env USER=testuser
dotfiles variables get paths.home --env USERPROFILE=C:\\test
```

### **Step-by-Step Debugging**

```bash
# 1. Check platform detection
dotfiles info

# 2. Validate configuration
dotfiles validate

# 3. Check variable sources
dotfiles variables sources

# 4. Test specific variable
dotfiles variables get problematic.variable

# 5. Trace variable source (both raw and processed)
dotfiles variables trace problematic.variable --raw
dotfiles variables trace problematic.variable

# 6. Test with overrides
dotfiles variables get problematic.variable --platform linux
```

### **File-by-File Debugging**

```bash
# Test each variable file individually
echo "Testing global.yaml:"
cat variables/global.yaml

echo "Testing platform file:"
cat variables/platforms/windows.yaml

echo "Testing variable loading:"
dotfiles variables sources
```

## ⚡ **Performance Issues**

### **Slow Variable Loading**

**Problem**: Commands take too long to execute

**Debug Steps**:

```bash
# 1. Check number of import files
dotfiles variables sources | wc -l

# 2. Look for complex templates
grep -r "{{.*}}" variables/

# 3. Check for large files
du -h variables/*.yaml
```

**Performance Tips**:
- Limit deep import chains
- Avoid complex template expressions
- Use conditional imports to reduce loaded files
- Keep variable files focused and small

### **Memory Usage**

```bash
# Monitor memory usage during variable loading
time dotfiles variables list
```

## 🆘 **Getting Help**

### **Self-Diagnosis Checklist**

Before asking for help, run this checklist:

```bash
# 1. Basic functionality
dotfiles --version
dotfiles info

# 2. Configuration validation
dotfiles validate

# 3. Variable system health
dotfiles variables sources
dotfiles variables list | head -10

# 4. Platform detection
dotfiles info | grep -E "(OS|Arch|Shell)"
```

### **Collecting Debug Information**

When reporting issues, include:

```bash
# System information
dotfiles info

# Configuration validation
dotfiles validate

# Variable sources
dotfiles variables sources

# Specific error reproduction
dotfiles variables get problematic.variable --verbose
```

### **Debug Information Template**

```
**System Information:**
```bash
dotfiles info
```

**Configuration:**
```bash
dotfiles validate
```

**Variable Sources:**
```bash
dotfiles variables sources
```

**Error Details:**
[Paste the exact error message and command that caused it]

**Expected vs Actual:**
- Expected: [What you expected to happen]
- Actual: [What actually happened]
```

### **Common Solutions Quick Reference**

| Problem | Quick Fix |
|---------|-----------|
| Command not found | Run `go install github.com/vleeuwenmenno/dotfiles-cp/cmd/dotfiles@latest` |
| Config not found | Run from dotfiles directory or use `dotfiles init` |
| Variable not found | Check `dotfiles variables sources` |
| Wrong paths | Use `pathJoin` function in templates |
| Import issues | Check for circular imports with `dotfiles validate` |
| Template errors | Verify Go template syntax |

---

**Need more help?** 
- Check [Variables Documentation](variables.md)
- Review [Configuration Reference](config-reference.md)  
- Open an issue on [GitHub](https://github.com/vleeuwenmenno/dotfiles-cp/issues)