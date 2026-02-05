# PowerShell Variable Escaping Fix

## 🐛 Issue

When running the Windows batch setup scripts, the following error occurred:

```
Setting up environment...
$($matches[1]) was unexpected at this time.
make: *** [Makefile:43: setup] Error 255
```

## 🔍 Root Cause

The error was caused by using inline PowerShell commands with the `-Command` parameter in batch files. The batch script interpreter was trying to process PowerShell variables like `$($matches[1])` before passing them to PowerShell, causing syntax errors.

### Problematic Code:
```batch
powershell -Command "$existing = Get-Content '.env' -Raw; ... $($matches[1]) ..."
```

The batch file sees `$` and tries to interpret it as a batch variable, leading to the error.

## ✅ Solution

Instead of using inline `-Command` with complex PowerShell code, we created a separate PowerShell script file that handles the logic.

### Created: `append-env-vars.ps1`

```powershell
param(
    [string]$TargetEnv = ".env",
    [string]$TemplateSource = ".env.template"
)

# Logic to append missing variables
# No escaping issues since it's in a proper .ps1 file
```

### Updated Batch Files:

**Before:**
```batch
powershell -Command "$existing = ... $($matches[1]) ..."
```

**After:**
```batch
powershell -File "build\scripts\append-env-vars.ps1" -TargetEnv ".env" -TemplateSource ".env.template"
```

## 📁 Files Modified

1. ✅ **setup.bat** - Fixed PowerShell invocation
2. ✅ **setup-y.bat** - Fixed PowerShell invocation
3. ✅ **common.bat** - Fixed PowerShell invocation
4. ✅ **append-env-vars.ps1** (NEW) - Extracted PowerShell logic

## 🎯 Benefits

### 1. **No More Escaping Issues**
- PowerShell variables are in a proper .ps1 file
- Batch doesn't try to interpret them
- Cleaner, more maintainable code

### 2. **Better Separation of Concerns**
- Batch handles batch logic
- PowerShell handles PowerShell logic
- Each script uses its native strengths

### 3. **Easier Debugging**
- PowerShell script can be run independently
- Easier to test and modify
- Better error messages

### 4. **Reusability**
- PowerShell script can be used by other scripts
- Single place to update logic
- Follows DRY principle

## 🧪 Testing

### Test 1: New .env Creation
```batch
cd build\scripts
setup.bat
```
**Expected**: Creates .env from template
**Result**: ✅ Works correctly

### Test 2: Existing .env with Missing Variables
```batch
REM Create partial .env
echo PUBLIC_HOST=localhost > .env

REM Run setup
setup.bat
```
**Expected**: Appends missing variables
**Result**: ✅ Works correctly

### Test 3: Existing .env with All Variables
```batch
REM Create complete .env
copy .env.template .env

REM Run setup
setup.bat
```
**Expected**: Reports all variables exist
**Result**: ✅ Works correctly

## 📖 Technical Details

### Why `-File` Instead of `-Command`?

| Aspect | `-Command` | `-File` |
|--------|-----------|---------|
| **Escaping** | Complex, error-prone | Simple, no escaping needed |
| **Debugging** | Difficult to debug | Easy to debug |
| **Maintenance** | Hard to read/modify | Easy to read/modify |
| **Reusability** | Not reusable | Fully reusable |
| **IDE Support** | No syntax highlighting | Full IDE support |

### PowerShell Script Structure

```powershell
# Parameters with defaults
param(
    [string]$TargetEnv = ".env",
    [string]$TemplateSource = ".env.template"
)

# Check if target exists
if (Test-Path $TargetEnv) {
    $existing = Get-Content $TargetEnv -Raw
} else {
    $existing = ""
}

# Read template
$template = Get-Content $TemplateSource

# Find missing variables
$missing = @()
foreach ($line in $template) {
    if ($line -match '^([A-Z_]+)=') {
        $varName = $matches[1]  # No escaping issues here!
        if ($existing -notmatch "(?m)^$varName=") {
            $missing += $line
        }
    }
}

# Append if needed
if ($missing.Count -gt 0) {
    Add-Content $TargetEnv "`n# Added by setup script on $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
    Add-Content $TargetEnv $missing
    Write-Host "[OK] Added $($missing.Count) missing variable(s) to $TargetEnv"
} else {
    Write-Host "[OK] All variables already exist in $TargetEnv"
}
```

## 🚀 Usage

### From Batch Files:
```batch
powershell -NoProfile -ExecutionPolicy Bypass -File "build\scripts\append-env-vars.ps1" -TargetEnv ".env" -TemplateSource ".env.template"
```

### Standalone:
```powershell
.\build\scripts\append-env-vars.ps1 -TargetEnv ".env" -TemplateSource ".env.template"
```

### With Defaults:
```powershell
.\build\scripts\append-env-vars.ps1
```

## 🔧 Future Improvements

- [ ] Add error handling for missing template file
- [ ] Add verbose mode for detailed output
- [ ] Add dry-run mode to preview changes
- [ ] Add validation for environment variable format
- [ ] Create similar scripts for other operations

## 📝 Lessons Learned

1. **Keep languages separate**: Don't mix complex PowerShell in batch scripts
2. **Use script files**: Easier to maintain and debug
3. **Proper parameters**: Use typed parameters in PowerShell
4. **Test edge cases**: Empty .env, missing .env, complete .env
5. **Document thoroughly**: Help future developers understand the solution

## ✨ Conclusion

The fix successfully resolves the PowerShell variable escaping issue by:
- Extracting PowerShell logic to a dedicated script file
- Using `-File` parameter instead of `-Command`
- Eliminating complex escaping requirements
- Improving code maintainability and reusability

All Windows batch setup scripts now work correctly without any escaping errors! 🎉
