# License Compliance Guide

This document explains SereniBase's license compliance strategy and how to handle FOSSA license scan issues.

## Project License

SereniBase is licensed under the **MIT License** - a permissive open-source license that allows:
- ✅ Commercial use
- ✅ Modification
- ✅ Distribution
- ✅ Private use
- ⚠️ With attribution required
- ⚠️ No liability/warranty

See [LICENSE](../LICENSE) for full terms.

## Dependency License Policy

### Allowed (✅ Green - No Action)
- **MIT** - Permissive, compatible with commercial use
- **Apache 2.0** - Permissive with patent protection
- **ISC** - Simple permissive license
- **BSD-2-Clause / BSD-3-Clause** - Permissive, BSD family
- **Unlicense** - Public domain dedication

### Requires Review (🟡 Yellow - Check & Override)
- **LGPL-2.0 / LGPL-2.1 / LGPL-3.0** - Weak copyleft
  - **Action:** Verify dynamic linking (safe for libraries linked at runtime)
  - **Override:** If only used as a dynamic library, safe to approve

### Must Replace (🔴 Red - Do Not Use)
- **GPL-2.0 / GPL-3.0** - Strong copyleft, requires source disclosure
- **AGPL-3.0** - Network copyleft, most restrictive
- **Action:** Replace with compatible alternative or get legal approval

## Understanding FOSSA Issues

### Denied (8 Issues - 🔴 Red)
**These block CI/CD:**
- Usually: GPL, AGPL, or other restrictive licenses
- Action: Replace the dependency or remove it
- Example:
  ```
  ❌ dependency-X@2.0.0 → GPL-2.0
  ✅ Use alternative-Y@1.5.0 → MIT
  ```

### Flagged (2 Issues - 🟡 Yellow)
**These need review:**
- Usually: LGPL, MPL, or licenses with special conditions
- Action: Review & either override or replace
- Common: False positives or overly cautious scans

## How to Fix Issues

### Step 1: Identify Offenders
1. Go to [FOSSA Dashboard](https://app.fossa.com)
2. Filter by:
   - Issues → "Denied"
   - Issues → "Flagged"
3. Note the package names and versions

### Step 2: Add Override in `.fossa.yml`

If you want to approve a license override (for known-safe packages):

```yaml
dependencies:
  # Safe LGPL library - used dynamically
  - name: some-safe-lgpl-lib
    version: v1.2.3
    license: LGPL-2.1
    # FOSSA will now treat this as approved

  # Node package with correct license
  - name: @scope/package-name
    version: 1.0.0
    license: MIT
```

### Step 3: Replace Problematic Dependencies

If a package has a restricted license, find an alternative:

```bash
# Example: Find alternatives to GPL package
# Old (GPL-2.0)
grep "old-package" go.mod package.json

# New (MIT)
go get github.com/alternative-package@latest
npm install alternative-package --save
```

### Step 4: Re-run Analysis

```bash
# Install FOSSA CLI
curl -H 'Cache-Control: no-cache' https://raw.githubusercontent.com/fossas/fossa-cli/master/install-latest.sh | bash

# Analyze with your .fossa.yml
fossa analyze --verbose

# Test against policy (blocks CI if violations)
fossa test --verbose
```

## FOSSA in CI/CD

We've integrated FOSSA into [.github/workflows/security-scan.yml](.github/workflows/security-scan.yml):

- **On every push** to `main` or `develop`
- **On every PR**
- **Daily schedule** at 2 AM UTC

### To Enable FOSSA in GitHub Actions:

1. Sign up at [fossa.com](https://fossa.com) (free tier available)
2. Connect your GitHub repo
3. Get your API key from FOSSA dashboard
4. Add GitHub Secret:
   - Go to Repo Settings → Secrets & Variables → Actions
   - Add `FOSSA_API_KEY` = your API key from FOSSA

Then FOSSA will:
- Automatically analyze dependencies
- Enforce your license policy
- Block PRs if violations exist

## Common Issues & Solutions

### "Unknown License" 🟡
**Cause:** Package metadata missing or license not declared
**Fix:**
```yaml
dependencies:
  - name: unlicensed-package
    version: 1.0.0
    license: MIT  # Override if you know the actual license
```

### "False Positive License" 🟡
**Cause:** FOSSA misdetected the license (common in comments/docs)
**Fix:**
```yaml
dependencies:
  - name: false-positive-package
    version: 2.1.0
    license: MIT  # Correct the detected license
```

### "GPL/AGPL License" 🔴
**Cause:** Package uses restricted copyleft license
**Fix:**
```bash
# Option 1: Remove if not critical
go mod edit -droprequire=github.com/gpl-package

# Option 2: Use compatible alternative
go get github.com/mit-alternative@latest
npm install --save mit-alternative
```

## Understanding Transitive Dependencies

⚠️ **Important:** Most license issues come from **transitive dependencies** (dependencies of your dependencies):

```
Your Code (MIT)
  ↓
  ├─ Library A (MIT)
  │   └─ Library X (GPL) ← Hidden problem!
  └─ Library B (Apache 2.0)
```

FOSSA catches these deep problems—you don't see them with `dep list`.

## Contributing with License Compliance

When adding new dependencies:

```bash
# Before adding
npm install new-package
go get github.com/new/package

# Check license
npm view new-package license
go mod graph | grep new-package

# Ensure it's compatible with our policy
# (MIT, Apache 2.0, ISC, BSD family)
```

## Legal Note

🔥 **IMPORTANT:** 
- FOSSA policy violation ≠ Legal violation
- Policy is preventive (best practices)
- If unsure about a license, **contact legal** before using
- This guide is policy, not legal advice

## Resources

- [MIT License](https://opensource.org/licenses/MIT)
- [FOSSA Documentation](https://docs.fossa.com)
- [OpenSource.org Licenses](https://opensource.org/licenses)
- [SPDX License List](https://spdx.org/licenses)

## Questions?

If you have questions about license compliance:
1. Check this guide
2. Review FOSSA dashboard
3. Open an issue with `license` label
4. Contact the Security team
