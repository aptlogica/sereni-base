# FOSSA License Compliance Quick Start

## 🚀 TL;DR - Before Adding Any Dependency

**Always check the license first:**

```bash
# For npm packages
npm view <package-name> license

# For Go packages  
go mod graph | grep <package-name>
# Or check repo: https://github.com/<org>/<repo>/blob/main/LICENSE
```

**Safe licenses (auto-pass):**
✅ MIT | ✅ Apache 2.0 | ✅ ISC | ✅ BSD-2 | ✅ BSD-3

**Need review:**
🟡 LGPL (usually OK, but verify) | 🟡 MPL

**DO NOT USE:**
❌ GPL | ❌ AGPL | ❌ Custom restrictive licenses

---

## 📋 Checklist Before Adding New Dependency

- [ ] Is the license visible in the repo? (Look for LICENSE file)
- [ ] License is in approved list? (MIT, Apache 2.0, etc.)
- [ ] Does it have transitive dependencies? (Check their licenses too)
- [ ] Run `npm audit` or `go mod tidy` to check for issues?

**If all ✅ → Safe to add**
**If any ❌ → Ask before adding**

---

## 🔧 When CI/CD FOSSA Check Fails

### Check what failed:
```bash
# View failures locally
fossa test --verbose
```

### Fix it quickly:

1. **If "Unknown License":**
   - Add override to `.fossa.yml` with correct license
   - Re-run: `fossa analyze && fossa test`

2. **If GPL/AGPL:**
   - Replace with alternative package
   - `go get <new-package>` or `npm install <new-package>`

3. **If "False Positive":**
   - Correct the license in `.fossa.yml`

---

## 📞 Examples

### Adding a new Go dependency
```bash
# 1. Install it
go get github.com/user/pkg@latest

# 2. Check license
curl https://raw.githubusercontent.com/user/pkg/main/LICENSE

# 3. If MIT/Apache → ✅ OK
# 4. If GPL → ❌ Remove and find alternative
```

### Adding a new npm package
```bash
# 1. Install it
npm install new-package

# 2. Check license
npm view new-package license

# 3. Same checks as Go
```

---

## ⚠️ What NOT To Do

❌ Ignore FOSSA warnings (they block CI)
❌ Use GPL packages without legal review
❌ Assume "permissive-sounding" licenses are safe
❌ Skip transitive dependency checks

---

## 🎯 Questions?

See the full guide: [docs/LICENSE_COMPLIANCE.md](../docs/LICENSE_COMPLIANCE.md)
