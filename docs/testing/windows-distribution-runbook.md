# zscalerctl — Windows 11 Distribution Landmine Runbook

> Goal: surface every Windows landmine that would make a normal finance colleague bounce off **before** distribution, so the maintainer never becomes the remote support desk. Run on a representative Windows 11 finance-lockdown host. Two agents (Claude + Codex) run this independently and post results from the same numbered scenarios.
>
> **Rules of engagement**
> - Run in a **NON-elevated, normal-user PowerShell** unless a scenario explicitly says "(elevated)". Elevation changes the file owner to `BUILTIN\Administrators` and silently changes results.
> - Test the **built `zscalerctl.exe`** only — never `go run` / `go test`. Those mask the real NTFS DACL the shipped artifact sees.
> - On any **permission denial or prompt that won't render** (remote session): STOP and report. Do NOT improvise workarounds.
> - Every reject message prints a **raw SID** (e.g. `S-1-5-32-545`), not a friendly name. The `Show-Acl` dump is how you map SID → principal. Always run `Show-Acl` BEFORE the command and paste both.

---

## 0. Prerequisites & Setup (run once)

### 0.1 Build the shipped-equivalent binary
```powershell
$env:CGO_ENABLED = '0'
go build -o .\zscalerctl.exe .\cmd\zscalerctl
# Sanity: must be a static PE32+ console exe, ~28MB
Get-Item .\zscalerctl.exe | Select-Object Name, Length
.\zscalerctl.exe version; "BUILD_EXIT=$LASTEXITCODE"
```
Expected: `version` exits 0; size ~28MB. If build fails, STOP — nothing downstream is interpretable.

### 0.2 Helper functions (paste all into the session)
```powershell
# Run zscalerctl and capture exit code + combined output exactly as a colleague sees it.
function Run-Z { param([string[]]$ZArgs)
  $out  = & .\zscalerctl.exe @ZArgs 2>&1 | Out-String
  $code = $LASTEXITCODE
  Write-Host "EXIT=$code"
  Write-Host "---OUT---"
  Write-Host $out
}

# Inspect the DACL the validator will see (maps the reject SID to a principal).
function Show-Acl { param([string]$p)
  Write-Host "--- icacls $p ---"; icacls $p
  Write-Host "OWNER = $((Get-Acl $p).Owner)"
}

# Make a file owner-only NTFS (break inheritance, grant only current user).
function Lock-OwnerOnly { param([string]$p)
  icacls $p /inheritance:r | Out-Null
  icacls $p /grant:r "$($env:USERNAME):(R,W)" | Out-Null
}

# Write an owner-only file with content in one shot.
function New-OwnerOnlyFile { param([string]$Path,[string]$Content)
  Set-Content -Path $Path -Value $Content -Encoding utf8
  Lock-OwnerOnly $Path
}

# Drive-type probe (0 UNKNOWN,1 NOROOT,2 REMOVABLE,3 FIXED,4 REMOTE,5 CDROM,6 RAMDISK)
Add-Type -Namespace W -Name K -MemberDefinition '[DllImport("kernel32.dll",CharSet=CharSet.Unicode)] public static extern uint GetDriveType(string root);' -ErrorAction SilentlyContinue
function DriveType { param([string]$root) [W.K]::GetDriveType($root) }
```

### 0.3 Reusable config bodies
```powershell
# Minimal valid profile (a permission ACCEPT then parses & prints; a REJECT never reaches parsing).
$BODY = "default_profile: dev`nprofiles:`n  dev:`n    auth_mode: oneapi`n    vanity_domain: example"
# Working dir for fixtures
$WORK = Join-Path $env:USERPROFILE 'zctl-test'; New-Item -ItemType Directory -Force $WORK | Out-Null
```

### 0.4 Record the host baseline (paste output verbatim into the report)
```powershell
cmd /c ver
"DomainJoined = " + (Get-CimInstance Win32_ComputerSystem).PartOfDomain
"APPDATA  = $env:APPDATA"
"LOCALAPPDATA = $env:LOCALAPPDATA"
"USERPROFILE = $env:USERPROFILE"
"APPDATA under OneDrive? " + ($env:APPDATA -match 'OneDrive')
$q = (Split-Path $env:USERPROFILE -Qualifier).TrimEnd(':')
try { "USERPROFILE drive DisplayRoot = " + (Get-PSDrive $q).DisplayRoot } catch { "no DisplayRoot" }
net use
net share
Get-MpComputerStatus | Select-Object RealTimeProtectionEnabled, IsTamperProtected
Get-ExecutionPolicy -List
```
> **Why this matters:** if `%APPDATA%`/`%USERPROFILE%` is redirected to a UNC/network home, or `%APPDATA%` is OneDrive-redirected, results may not generalize — call it out loudly. Folder redirection to a network home is the single most likely mass false-reject vector.

### Exit-code contract (verified in `cmd/zscalerctl/main.go`)
`0` success · `1` internal · `2` usage/invalid_config · `3` credential/missing_credentials · `4` not_found · `5` live_access_failed · `6` partial_dump · `7` drift.

> **Verified behavior corrections folded into this runbook** (read these — several sub-plan guesses were wrong):
> - Config-file insecure-perms → `config.ErrInvalidConfig` → **exit 2**, kind `invalid_config` (eager, at load).
> - **`file:` secret insecure-perms surfaces at RESOLVE time as exit 3**, kind `missing_credentials` — NOT exit 1. The resolver error is wrapped at `internal/cli/app.go:1207`: `"%w: resolve client secret: %w"` with `zscaler.ErrMissingCredentials`. Earlier guesses of exit-1/"internal" for secret rejects are WRONG.
> - `config show`, `doctor`, `auth status` **never resolve secrets** — they report `credentials: configured` / `live_api: available` from scheme-presence (`Configured()`/`IsConfigured()`) alone (`internal/cli/app.go` `credentialStatus`/`liveAPIStatus`). A wired-but-broken secret shows green there. Only a real read command (`zia .../zpa ...`/`dump`) resolves.
> - The over-redaction defect is confirmed in source (`internal/redact/redact.go:326-331`, rule `secret_phrase`): `(?i)\b(?:...secret...)\s+([A-Za-z0-9._~+/=|:-]{8,})\b` → marker. It eats the diagnostic verb after "secret"/"credential" in error chains.

---

## SURFACE A — Owner-only DACL validation on LOCAL NTFS (the #1 distribution risk)

### A1 — Default %APPDATA% via Set-Content (GO/NO-GO)
**Intent:** the happy path that decides distribution — a config made the normal way at the default location must be ACCEPTED with zero ceremony and via default discovery (no `--config`).
```powershell
$dir = Join-Path $env:APPDATA 'zscalerctl'; New-Item -ItemType Directory -Force $dir | Out-Null
$cfg = Join-Path $dir 'config.yaml'; Remove-Item $cfg -ErrorAction SilentlyContinue
Set-Content -Path $cfg -Value $BODY -Encoding utf8
Show-Acl $cfg
Run-Z @('--format','table','config','show')
```
**Expected:** EXIT=0; table shows Profile=dev, Config=the file path, Auth Mode=oneapi. No "unsafe file permissions" line. Proves both the accept-set and `%APPDATA%` default discovery.
**Failure means:** CATASTROPHIC — every colleague is blocked on first run. The inherited `%APPDATA%` DACL ({owner, SYSTEM, Administrators}) is being rejected; inspect which SID fired (likely an inherited `BUILTIN\Users`/`Authenticated Users` from a GPO-modified profile). Code-level blocker, not user error.
**Report:** EXIT, full stdout, icacls dump + owner, ACCEPT/REJECT. If REJECT: the exact SID and the icacls line it maps to. **This single result is the go/no-go signal.**

### A2 — Default %APPDATA% via Notepad (the real colleague workflow)
**Intent:** GUI atomic-save can re-inherit a different DACL than scripted creation; also catches the `config.yaml.txt` footgun.
```powershell
$dir = Join-Path $env:APPDATA 'zscalerctl'; New-Item -ItemType Directory -Force $dir | Out-Null
$cfg = Join-Path $dir 'config.yaml'; Remove-Item $cfg -ErrorAction SilentlyContinue
Write-Host "In Notepad paste the dev profile, then Save As -> $cfg (Save as type: All Files, NOT .txt)"
notepad $cfg
Read-Host 'Press Enter after saving and closing Notepad'
Get-ChildItem $dir   # confirm filename is config.yaml, not config.yaml.txt
Show-Acl $cfg
Run-Z @('--format','table','config','show')
```
**Expected:** EXIT=0, config table. File is `config.yaml` (NOT `.txt`).
**Failure means:** If REJECT where A1 ACCEPTed → Notepad's save path yields a different DACL (stealth false-reject for the exact workflow colleagues use). If saved as `.txt` → default discovery finds nothing, exit 0 with Config not loaded / env-only (a silent no-op trap worth documenting).
**Report:** ACCEPT/REJECT + EXIT + icacls; the saved filename (was `.txt` appended?); whether GUI-save DACL matches A1.

### A3 — %USERPROFILE%\.config via --config
**Intent:** different inheritance source than `%APPDATA%`; on Windows this is not auto-discovered (only via `--config`).
```powershell
$dir = Join-Path $env:USERPROFILE '.config\zscalerctl'; New-Item -ItemType Directory -Force $dir | Out-Null
$cfg = Join-Path $dir 'config.yaml'; Remove-Item $cfg -ErrorAction SilentlyContinue
Set-Content -Path $cfg -Value $BODY -Encoding utf8
Show-Acl $cfg
Run-Z @('--config',$cfg,'--format','table','config','show')
```
**Expected:** EXIT=0, config table. Proves `--config` works and profile-root inheritance passes.
**Failure means:** REJECT here but ACCEPT in A1 → inheritance differs between `%APPDATA%` and the profile root in this image; documenting `.config` would also require a HOME-based default (currently not discovered).
**Report:** EXIT + icacls + owner; ACCEPT/REJECT + SID if REJECT; note this path is only reachable via `--config` on Windows.

### A4 — DAV-29 disposition: %TEMP% vs %APPDATA% (resolve the open bug)
**Intent:** decide whether the DAV-29 reject is a temp-dir/`t.TempDir` artifact or a real-location bug.
```powershell
$tmp = Join-Path $env:TEMP ('zctl_'+[guid]::NewGuid().ToString('N')); New-Item -ItemType Directory -Force $tmp | Out-Null
$tcfg = Join-Path $tmp 'config.yaml'; Set-Content -Path $tcfg -Value $BODY -Encoding utf8
Write-Host '=== TEMP location ==='; Show-Acl $tcfg
Run-Z @('--config',$tcfg,'--format','table','config','show')
$acfg = Join-Path $env:APPDATA 'zscalerctl\config.yaml'
Write-Host '=== APPDATA location ==='; Show-Acl $acfg
Run-Z @('--config',$acfg,'--format','table','config','show')
```
**Expected:** likely TEMP=REJECT exit 2 (inherited `BUILTIN\Users` S-1-5-32-545, message `inherited non-owner principal S-1-5-32-545 has read/write access`) BUT APPDATA=ACCEPT exit 0. That split confirms DAV-29 is a temp-dir artifact.
**Failure means:** if APPDATA also REJECTs with the same SID → DAV-29 is a REAL distribution blocker → code fix required. If only TEMP rejects → downgrade to "never run from %TEMP%, never trust t.TempDir parity."
**Report:** both EXITs, both icacls dumps side by side, exact reject SID + message, and verdict: artifact vs real bug. Note which branch fired — inherited (`fileperm_windows.go:99`) vs non-inherited (`:101`).

### A5 — Grant BUILTIN\Users (classic "copied to a shared spot")
**Intent:** must REJECT and name the offending principal; gauge actionability.
```powershell
$cfg = Join-Path $env:APPDATA 'zscalerctl\config.yaml'
Set-Content -Path $cfg -Value $BODY -Encoding utf8
icacls $cfg /grant 'BUILTIN\Users:(R)'
Show-Acl $cfg
Run-Z @('--format','table','config','show')
Run-Z @('--format','json','config','show')
```
**Expected:** REJECT exit 2; text `zscalerctl: invalid config: config file permissions: unsafe file permissions: broad Windows principal S-1-5-32-545 has read/write access` (non-inherited grant → `sidIsBlocked` → "broad Windows principal" branch, `fileperm_windows.go:93`). JSON kind=invalid_config. **GAP:** message prints `S-1-5-32-545`, not `BUILTIN\Users`, and offers no `icacls` fix.
**Failure means:** ACCEPT → blocklist misses a directly-granted Users ACE (secret-bearing config readable by all local users). REJECT-but-unparseable → support-burden finding.
**Report:** EXIT, exact text + JSON verbatim, the SID, and a judgment: could a non-expert self-fix from this message alone? (Likely NO — flag friendly-name + icacls-hint.)

### A6 — Grant Everyone (S-1-1-0) and Authenticated Users (S-1-5-11)
**Intent:** confirm each blocked well-known SID rejects; check message distinguishability.
```powershell
$cfg = Join-Path $env:APPDATA 'zscalerctl\config.yaml'
foreach ($p in @('Everyone','Authenticated Users')) {
  Set-Content -Path $cfg -Value $BODY -Encoding utf8
  icacls $cfg /inheritance:r /grant:r "$($env:USERNAME):(F)" | Out-Null
  icacls $cfg /grant "${p}:(R)"
  Write-Host "=== granted: $p ==="; Show-Acl $cfg
  Run-Z @('--format','table','config','show')
}
```
**Expected:** BOTH REJECT exit 2. Everyone → `broad Windows principal S-1-1-0 ...`; Authenticated Users → `broad Windows principal S-1-5-11 ...`.
**Failure means:** any ACCEPT is a world-readable-secret miss. Opaque SIDs reinforce A5's actionability gap.
**Report:** per-principal EXIT + exact message + SID. Are all blocked SIDs caught and messages distinguishable/actionable?

### A7 — Grant a domain group (GPO-shape false-reject probe)
**Intent:** blocked `Domain Users` (RID 513) vs an arbitrary domain group (generic non-owner). This is the highest-probability enterprise false-reject (GPO-applied group ACEs).
```powershell
$cfg = Join-Path $env:APPDATA 'zscalerctl\config.yaml'
# Domain Users half
Set-Content -Path $cfg -Value $BODY -Encoding utf8
icacls $cfg /inheritance:r /grant:r "$($env:USERNAME):(F)" | Out-Null
icacls $cfg /grant 'Domain Users:(R)'; Show-Acl $cfg
Run-Z @('--format','table','config','show')
# Arbitrary finance group half — REPLACE <DOMAIN>\<group> with a real resolvable group; skip if none.
Set-Content -Path $cfg -Value $BODY -Encoding utf8
icacls $cfg /inheritance:r /grant:r "$($env:USERNAME):(F)" | Out-Null
icacls $cfg /grant '<DOMAIN>\<a-finance-group>:(R)'; Show-Acl $cfg
Run-Z @('--format','table','config','show')
```
**Expected:** Domain Users → REJECT exit 2 `broad Windows principal S-1-5-21-...-513 ...`. Arbitrary group → REJECT exit 2 `non-owner principal S-1-5-21-...-<rid> ...` (`fileperm_windows.go:101`). Both print SIDs, not names.
**Failure means:** an arbitrary group ACCEPTing would be wrong; rejecting is correct-but-strict — meaning ANY GPO-applied group grant becomes a false-reject. Key design signal.
**Report:** both EXITs + messages + SIDs. State whether this image applies any default group ACE via GPO that would hit innocent files. If no usable group, run only the Domain Users half and say so.

### A8 — Owner identity: current-user vs Administrators (IT-pushed-config model)
**Intent:** can an admin-deployed (Administrators-owned) config be READ by the colleague?
```powershell
$cfg = Join-Path $env:APPDATA 'zscalerctl\config.yaml'
Set-Content -Path $cfg -Value $BODY -Encoding utf8
icacls $cfg /inheritance:r /grant:r "$($env:USERNAME):(F)" | Out-Null
Write-Host '=== owner = current user ==='; Show-Acl $cfg
Run-Z @('--format','table','config','show')
Write-Host '=== set owner = Administrators (elevates) ==='
Start-Process -Verb RunAs -Wait powershell -ArgumentList '-NoProfile','-Command',"icacls '$cfg' /setowner 'BUILTIN\Administrators'; icacls '$cfg' /inheritance:r /grant:r 'BUILTIN\Administrators:(F)' '$($env:USERNAME):(R)'"
Show-Acl $cfg
Run-Z @('--format','table','config','show')
```
**Expected:** (a) owner=user → ACCEPT exit 0. (b) owner=Administrators with only Administrators+owner-user ACEs → ACCEPT exit 0 (Administrators is in `windowsAllowedSIDs` and not in `sidIsBlocked`).
**Failure means:** if owner=Administrators REJECTs, an IT-pushed locked-down config is unusable → deployment-model blocker. If ACCEPT → IT-managed rollout is viable (lowest support burden).
**Report:** both EXITs + owner + icacls. Implication for an IT-pushed-config model.

### A9 — Inheritance ON (loose parent) vs broken-inheritance owner-only (proves the remediation one-liner)
**Intent:** isolate the inherited-ACE rule and verify the exact `icacls` fix to ship in the error text.
```powershell
$base = Join-Path $env:APPDATA 'zctl_inh'; Remove-Item $base -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $base | Out-Null
icacls $base /grant 'BUILTIN\Users:(OI)(CI)(R)'
$cfg = Join-Path $base 'config.yaml'; Set-Content -Path $cfg -Value $BODY -Encoding utf8
Write-Host '=== (c) inherited Users from parent ==='; Show-Acl $cfg
Run-Z @('--config',$cfg,'--format','table','config','show')
Write-Host '=== (b) break inheritance, owner-only ==='
icacls $cfg /inheritance:r /grant:r "$($env:USERNAME):(F)"; Show-Acl $cfg
Run-Z @('--config',$cfg,'--format','table','config','show')
```
**Expected:** (c) → REJECT exit 2 `inherited non-owner principal S-1-5-32-545 ...` (`:99`, distinct wording). (b) → ACCEPT exit 0. Proves `icacls <file> /inheritance:r /grant:r %USERNAME%:F` converts a rejected file into an accepted one.
**Failure means:** (c) ACCEPT → inherited-ACE rule not firing (loose parent silently exposes the file). (b) still REJECT → the published remediation does NOT work on this host (serious docs/UX failure, since it's the natural fix).
**Report:** both EXITs + icacls + messages. **State definitively: does `icacls /inheritance:r /grant:r %USERNAME%:F` fix a rejected file?** That one-liner is the remediation to ship in the error text and README.

---

## SURFACE B — NON-LOCAL volumes (UNC, mapped drive, OneDrive-redirected, removable, FAT)

> The validator reads ONLY the NTFS DACL via `GetSecurityInfo`/`GetNamedSecurityInfo`. It does **no** volume-type check and **cannot** see SMB share-level ACLs. Two failure directions: FALSE-ACCEPT (NTFS owner-only but share=Everyone:Read) and FALSE-REJECT (SID won't resolve over SMB).

### B1 — Baseline: local FIXED volume (reference accept)
```powershell
$dir = Join-Path $env:USERPROFILE 'zsctl-local'; New-Item -ItemType Directory -Force $dir | Out-Null
$cfg = Join-Path $dir 'config.yaml'; Set-Content -Path $cfg -Value $BODY -Encoding utf8; Lock-OwnerOnly $cfg
"DriveType=" + (DriveType (Split-Path -Qualifier $cfg))
Run-Z @('--config',$cfg,'config','show')
```
**Expected:** EXIT=0; DriveType=3 (FIXED). If this fails, the helper or validator is wrong — fix before proceeding.
**Report:** EXIT, DriveType, did config show render.

### B2 — (elevated) Loopback share with Everyone:Read at SHARE, owner-only at NTFS
```powershell
$shareBase = Join-Path $env:SystemDrive 'zsctltest'; New-Item -ItemType Directory -Force $shareBase | Out-Null
$cfg = Join-Path $shareBase 'config.yaml'; Set-Content -Path $cfg -Value $BODY -Encoding utf8; Lock-OwnerOnly $cfg
net share zsctltest=$shareBase /GRANT:Everyone,READ
net share zsctltest
```
**Expected:** share created (Everyone READ at share level); NTFS DACL on config.yaml stays owner-only. If `net share` needs elevation and is denied: STOP and report — do not improvise.
**Report:** share created? exact share-permission line; confirm NTFS stayed owner-only; elevation required?

### B3 — UNC soundness gap (headline FALSE-ACCEPT probe)
```powershell
$unc = "\\localhost\zsctltest\config.yaml"; Test-Path $unc
Run-Z @('--config',$unc,'config','show')
```
**Expected (observed-behavior probe):** most likely EXIT=0 and config show renders — the tool ACCEPTS a file Everyone can read over the share. If it rejects, capture the message (could be SMB SID false-reject — see B7).
**Failure means:** EXIT=0 is the CONFIRMED soundness gap: a "secured" config is world-readable on the file server. Strong evidence to refuse non-local volumes outright.
**Report:** EXIT, full message, explicit verdict: FALSE-ACCEPT despite Everyone:Read? **Primary design driver.**

### B4 — Mapped network drive (lettered remote hides its network nature)
```powershell
net use Z: \\localhost\zsctltest
"DriveType Z = " + (DriveType 'Z:\')
Run-Z @('--config','Z:\config.yaml','config','show')
net use Z: /delete /y
```
**Expected:** DriveType=4 (REMOTE); behavior mirrors B3 (likely EXIT=0). Point: `Z:\` looks local but is remote.
**Failure means:** accept gives false assurance for creds on a corporate home drive; detection must use `GetDriveType==4`, not just a `\\` prefix.
**Report:** DriveType for `Z:\`, EXIT, message.

### B5 — OneDrive-redirected default %APPDATA% (DriveType won't catch it)
```powershell
"APPDATA=$env:APPDATA"
$d = Join-Path $env:APPDATA 'zscalerctl'; New-Item -ItemType Directory -Force $d | Out-Null
$cfg = Join-Path $d 'config.yaml'; Set-Content -Path $cfg -Value $BODY -Encoding utf8; Lock-OwnerOnly $cfg
"OneDrive-redirected? " + ($env:APPDATA -match 'OneDrive')
"DriveType = " + (DriveType (Split-Path -Qualifier $cfg))
Run-Z @('config','show')
```
**Expected:** EXIT=0 from default path. DriveType usually 3 (FIXED) even when OneDrive-synced (local cache on C:). So DriveType ALONE will NOT catch OneDrive.
**Failure means:** if APPDATA is OneDrive-redirected and accepted, creds/refs sync to the cloud and to every machine the user signs into — a finance egress concern a pure DriveType check can't detect.
**Report:** is APPDATA under OneDrive? DriveType? EXIT? Note the gap: FIXED yet cloud-synced.

### B6 — Removable/USB (DriveType=2)
```powershell
$rv = (Get-Volume | Where-Object DriveType -eq 'Removable' | Select-Object -First 1).DriveLetter
if ($rv) { $root = "$rv`:\"; "DriveType = " + (DriveType $root); $cfg = "$root`zsctl-config.yaml"
  Set-Content -Path $cfg -Value $BODY -Encoding utf8; Lock-OwnerOnly $cfg
  "FileSystem = " + (Get-Volume -DriveLetter $rv).FileSystem
  Run-Z @('--config',$cfg,'config','show')
} else { Write-Host 'NO REMOVABLE VOLUME — report skipped' }
```
**Expected:** if present, DriveType=2, likely EXIT=0 (many USB are FAT/exFAT → no DACL, see B8). Else SKIP.
**Failure means:** accept on removable = creds can ride out of the building with the tool's blessing.
**Report:** DriveType, filesystem, EXIT, or "skipped".

### B7 — SMB SID false-reject probe
```powershell
$unc = "\\localhost\zsctltest\config.yaml"; icacls $unc
Run-Z @('--config',$unc,'config','show')
```
**Expected:** owner SID should match local token SID and ACCEPT. A false-reject surfaces as `config file permissions: unsafe file permissions: inherited/non-owner principal <SID> ...` exit 2 for a file that IS owner-only locally.
**Failure means:** false-reject = a perfectly-locked network-home file refused with a confusing perms error (support-desk magnet). Either direction → conclusion: don't validate DACLs over SMB; refuse the volume.
**Report:** EXIT, full message + any SID, accept or false-reject. Pair with B3.

### B8 — (elevated) FAT/exFAT (no DACL at all)
```powershell
$rv = (Get-Volume | Where-Object FileSystem -in 'FAT','FAT32','exFAT' | Select-Object -First 1).DriveLetter
if (-not $rv) {
  $vhd = Join-Path $env:TEMP 'zsctlfat.vhdx'
  @"
create vdisk file="$vhd" maximum=64
attach vdisk
create partition primary
format fs=fat32 quick label=ZSCTLFAT
assign letter=Y
exit
"@ | diskpart
  $rv = 'Y'
}
$root = "$rv`:\"; "DriveType=" + (DriveType $root); "FS=" + (Get-Volume -DriveLetter $rv).FileSystem
$cfg = "$root`config.yaml"; Set-Content -Path $cfg -Value $BODY -Encoding utf8
Run-Z @('--config',$cfg,'config','show')
# cleanup if VHD was created
if (Test-Path (Join-Path $env:TEMP 'zsctlfat.vhdx')) {
  @"
select vdisk file="$(Join-Path $env:TEMP 'zsctlfat.vhdx')"
detach vdisk
exit
"@ | diskpart
  Remove-Item (Join-Path $env:TEMP 'zsctlfat.vhdx') -Force
}
```
**Expected:** FS=FAT32; `GetNamedSecurityInfo` on FAT typically returns NULL/absent DACL → validator should hit its empty/NULL-DACL branch (exit 2) OR false-accept. Capture which. If diskpart needs elevation and is denied: STOP and report.
**Failure means:** ACCEPT on FAT = owner-only guarantee silently void on non-NTFS media → supports a filesystem-type (NTFS-required) check.
**Report:** FS type, EXIT, exact message (NULL-DACL branch vs accept).

### B9 — file: secret on UNC (second, lazy gate; exit 3 corrected)
```powershell
$local = Join-Path $env:USERPROFILE 'zsctl-local\config.yaml'
Set-Content -Path $local -Encoding utf8 -Value "default_profile: dev`nprofiles:`n  dev:`n    auth_mode: oneapi`n    vanity_domain: example`n    client_id: testid`n    client_secret_ref: file:\\localhost\zsctltest\client_secret.txt"
Lock-OwnerOnly $local
$sec = Join-Path $env:SystemDrive 'zsctltest\client_secret.txt'
Set-Content -Path $sec -Value 'supersecret' -Encoding ascii; Lock-OwnerOnly $sec
Write-Host '=== config show (lazy — should NOT resolve secret) ==='
Run-Z @('--config',$local,'config','show')
Write-Host '=== doctor (also does NOT resolve — reports configured from scheme) ==='
Run-Z @('--config',$local,'doctor')
Write-Host '=== live read forces resolution over UNC ==='
Run-Z @('--config',$local,'zia','locations','list')
```
**Expected:** config show & doctor EXIT=0 (lazy; report secret source `file`, never touch the UNC file). The live read resolves the `file:` ref over UNC: same soundness/identity behavior as B3/B7, wrapped via `credentials.ErrUnsafePermissions` → `resolve client secret:` → **exit 3, kind missing_credentials** (NOT exit 1 — corrected from the sub-plan).
**Failure means:** the network gap applies to secret files too. If the message is even less actionable than the config one, flag it (secret rejects appear later, at command run). Note that `doctor` cannot pre-flight this.
**Report:** config show & doctor EXIT/secret-source; live-read EXIT + exact message + kind; confirm lazy-vs-eager split.

### B10 — keyring is volume-independent (don't let a volume rule touch it)
```powershell
$cfg = Join-Path $env:USERPROFILE 'zsctl-local\config.yaml'
Set-Content -Path $cfg -Encoding utf8 -Value "default_profile: dev`nprofiles:`n  dev:`n    auth_mode: oneapi`n    vanity_domain: example`n    client_id: testid`n    client_secret_ref: keyring:zscalerctl/client_secret"
Lock-OwnerOnly $cfg
Run-Z @('--config',$cfg,'config','show')
Run-Z @('--config',$cfg,'zia','locations','list')
```
**Expected:** config show EXIT=0 (source keyring, lazy). Live read either resolves from Credential Manager (if stored) or returns the keyring not-found message — no file/volume logic, regardless of where config.yaml lives.
**Failure means:** any file/volume error here would be a regression if a future non-local-volume rule interfered with keyring.
**Report:** both EXITs; confirm no file/volume error appeared.

---

## SURFACE C — Shipped binary executing for a locked-down (possibly non-admin) colleague

### C1 — version / help with zero config (pre-LoadConfig)
```powershell
Run-Z @('version'); Run-Z @('--help'); Run-Z @('help')
```
**Expected:** EXIT=0 for all three; version prints a version/commit line; help lists commands. No prompt, admin, network, or stack trace.
**Failure means:** binary fundamentally broken on this build (missing runtime dep, AV quarantine on first exec) — hard stop.
**Report:** EXIT + first 2 lines each; anything prompted/blocked on first execution.

### C2 — Offline command set with no config present (LoadConfig clean-empty path)
```powershell
if (Test-Path (Join-Path $env:APPDATA 'zscalerctl\config.yaml')) { Write-Host 'PRECOND: move %APPDATA%\zscalerctl\config.yaml aside first' }
Run-Z @('schema','list'); Run-Z @('config','show'); Run-Z @('doctor'); Run-Z @('auth','status')
```
**Expected:** EXIT=0 for all. config show = env/unset source; auth status = not configured; doctor reports without a live call; schema list enumerates. No admin/network/crash. (A non-existent DEFAULT path is silently ignored — `config/profile.go:50`.)
**Failure means:** an unconfigured colleague bounces immediately, or default-path resolution misbehaves.
**Report:** EXIT per command; config show's reported source; confirm doctor made no network call; paste any non-zero stderr.

### C3 — Machine-first JSON off a non-TTY pipe
```powershell
$j = & .\zscalerctl.exe --format json schema list | Out-String; "EXIT=$LASTEXITCODE"
try { $j | ConvertFrom-Json | Out-Null; 'schema list JSON parses' } catch { 'schema list JSON BROKEN' }
Run-Z @('--format','json','config','show')
Run-Z @('--format','json','doctor')
```
**Expected:** EXIT=0; well-formed JSON each; config show JSON is the redacted Safe() view (no secrets).
**Failure means:** malformed/interleaved JSON breaks agent/automation consumers — undermines the machine-first promise.
**Report:** does ConvertFrom-Json succeed; any secret-looking values in config show JSON; EXIT per command.

### C4 — Run from a UAC/system-protected cwd
```powershell
Push-Location C:\Windows\System32; Run-Z @('version'); Run-Z @('schema','list'); Pop-Location
Push-Location 'C:\Program Files'; Run-Z @('config','show'); Pop-Location
```
**Expected:** EXIT=0 from each restricted cwd (read-only; writes nothing to cwd).
**Failure means:** failure only from a protected cwd → the binary writes to cwd (temp/cache/log) — a landmine; would force a "run from your home dir" caveat.
**Report:** EXIT per cwd; any file created in cwd (diff a dir listing before/after).

### C5 — (elevated copy, unelevated run) exe in Program Files
```powershell
# one-time elevated:
Start-Process -Verb RunAs -Wait powershell -ArgumentList '-NoProfile','-Command',"New-Item -ItemType Directory -Force 'C:\Program Files\zscalerctl' | Out-Null; Copy-Item '$((Resolve-Path .\zscalerctl.exe).Path)' 'C:\Program Files\zscalerctl\' -Force"
& 'C:\Program Files\zscalerctl\zscalerctl.exe' version; "EXIT=$LASTEXITCODE"
& 'C:\Program Files\zscalerctl\zscalerctl.exe' schema list | Out-Null; "EXIT=$LASTEXITCODE"
```
**Expected:** EXIT=0 unelevated (execute needs no write).
**Failure means:** AppLocker/WDAC default rules or a deny-execute policy blocking it → predicts colleague bounce. Distinguish "policy blocked" from "binary broke" by the error.
**Report:** EXIT; exact error if blocked; whether it's AppLocker (Event Log `Microsoft-Windows-AppLocker`) vs SmartScreen vs Defender.

### C6 — Mark-of-the-Web (downloaded exe simulation)
```powershell
Copy-Item .\zscalerctl.exe "$env:TEMP\zscalerctl-dl.exe" -Force
Set-Content -Path "$env:TEMP\zscalerctl-dl.exe" -Stream Zone.Identifier -Value "[ZoneTransfer]`r`nZoneId=3"
Get-Content "$env:TEMP\zscalerctl-dl.exe" -Stream Zone.Identifier
& "$env:TEMP\zscalerctl-dl.exe" version; "EXIT=$LASTEXITCODE"
```
**Expected:** MotW alone usually does NOT block a CLI exe from a console (SmartScreen App Reputation mainly gates Explorer double-click). Likely EXIT=0; Defender may scan on first run.
**Failure means:** if the MotW-stamped unsigned exe is blocked/quarantined, every downloader bounces → strongest argument for Authenticode code-signing.
**Report:** ZoneId=3 present? EXIT via console; `Get-MpThreatDetection` hit? does Explorer double-click differ from console launch? was `Unblock-File` needed?

### C7 — Defender real-time / cloud reaction to a low-prevalence Go exe
```powershell
Get-MpComputerStatus | Select-Object RealTimeProtectionEnabled, IsTamperProtected
Run-Z @('version'); Start-Sleep 3
Get-MpThreatDetection | Where-Object { $_.Resources -match 'zscalerctl' }
Test-Path .\zscalerctl.exe
```
**Expected:** no detection; EXIT=0; exe still present.
**Failure means:** false-positive quarantine = scary AV alert + vanished file → adoption killer; drives signing + Microsoft submission.
**Report:** RealTimeProtection state; any detection naming zscalerctl; exe still exists?

### C8 — No-toolchain analog (zero runtime deps)
```powershell
$saved = $env:Path; $env:Path = 'C:\Windows\System32;C:\Windows'
Get-Command go -ErrorAction SilentlyContinue
& .\zscalerctl.exe version; "EXIT=$LASTEXITCODE"
& .\zscalerctl.exe schema list | Select-Object -First 1; "EXIT=$LASTEXITCODE"
$env:Path = $saved
```
**Expected:** `go` not found on trimmed PATH, yet zscalerctl.exe runs EXIT=0.
**Failure means:** binary not self-contained (shelling out) → breaks "drop one exe and go". **ANALOG LIMIT:** the dev host HAS Go; trimming PATH is an approximation. The faithful test is a machine that never had Go — flag this.
**Report:** confirm `Get-Command go` returned nothing; EXIT of version + schema list; note Go-equipped host vs clean box.

### C9 — Per-user no-admin install + PATH
```powershell
$dst = Join-Path $env:LOCALAPPDATA 'Programs\zscalerctl'; New-Item -ItemType Directory -Force $dst | Out-Null
Copy-Item .\zscalerctl.exe $dst -Force
[Environment]::SetEnvironmentVariable('Path', ([Environment]::GetEnvironmentVariable('Path','User') + ';' + $dst), 'User')
$env:Path += ';' + $dst
zscalerctl version; "EXIT=$LASTEXITCODE"
Get-Command zscalerctl | Select-Object Source
```
**Expected:** EXIT=0; `zscalerctl` resolves from per-user Programs without admin.
**Failure means:** policy strips user PATH edits or execute-denies per-user dirs → install friction for non-admins.
**Report:** EXIT; Get-Command Source; did the User PATH edit persist to a new shell? any policy reverting it?

### C10 — Execution Policy irrelevance (pre-empt false reports)
```powershell
Get-ExecutionPolicy -List
& .\zscalerctl.exe version; "EXIT=$LASTEXITCODE"
```
**Expected:** EXIT=0 regardless of policy (the exe is launched directly, not as a script).
**Failure means:** if affected, surprising (implies a wrapper script) — investigate.
**Report:** the policy values; EXIT confirming the exe ran irrespective of policy.

### C11 — Crash resilience: malformed + oversized config
```powershell
$bad = Join-Path $env:TEMP 'zctl-bad.yaml'; Set-Content $bad 'this: is: not: valid: yaml: : :' -Encoding utf8
icacls $bad /inheritance:r /grant:r "$($env:USERNAME):(R)" | Out-Null
Run-Z @('--config',$bad,'config','show')
fsutil file createnew "$env:TEMP\zctl-big.yaml" 1100000 | Out-Null
icacls "$env:TEMP\zctl-big.yaml" /inheritance:r /grant:r "$($env:USERNAME):(R)" | Out-Null
Run-Z @('--config',"$env:TEMP\zctl-big.yaml",'config','show')
```
**Expected:** malformed → EXIT=2 `invalid config: parse config file`. Oversized (>1MB) → EXIT=2 `invalid config: read config file: config file exceeds 1048576 bytes`. No EXIT=1/panic, no WER dialog.
**Failure means:** exit-1/panic/crash dialog on bad input = support event + poor first impression.
**Report:** EXIT each (must be 2, not 1); verbatim stderr; confirm no crash/WER dialog.

---

## SURFACE D — Secret-provider matrix + error-message UX (env / file / cmd / keyring)

> Architecture (verified): ref **syntax** errors surface eagerly at `config show` (exit 2, invalid_config). Ref **resolution** errors are **lazy** — only at a live read (`zia .../zpa ...`/`dump`), wrapped at `internal/cli/app.go:1207` as `resolve client secret: <err>` under `ErrMissingCredentials` → **exit 3**. With no real tenant, a successful resolution surfaces as **exit 5** (live_access_failed) — that is the "secret is good" signal.

### D1 — env: happy (resolution succeeds → exit 5)
```powershell
New-OwnerOnlyFile (Join-Path $WORK 'env.yaml') "profiles:`n  default:`n    auth_mode: oneapi`n    vanity_domain: example`n    client_id: cid`n    client_secret_ref: env:ZS_SECRET"
$env:ZS_SECRET = 'dummy-value-1234'
Run-Z @('--config',(Join-Path $WORK 'env.yaml'),'zia','locations','list')
Run-Z @('--config',(Join-Path $WORK 'env.yaml'),'config','show','--format','json')
Remove-Item Env:ZS_SECRET
```
**Expected:** live read EXIT=5, kind=live_access_failed (`zscaler API request failed`). NOT exit 3 — proves env resolved. config show JSON shows `client_secret_scheme:"env"`.
**Failure means:** exit 3 → env resolution broken on Windows (LookupEnv/casing).
**Report:** EXIT + message; confirm scheme `env` in config show.

### D2 — env: unset (does it name the variable?)
```powershell
New-OwnerOnlyFile (Join-Path $WORK 'envmiss.yaml') "profiles:`n  default:`n    auth_mode: oneapi`n    vanity_domain: example`n    client_id: cid`n    client_secret_ref: env:ZS_NOT_SET"
Run-Z @('--config',(Join-Path $WORK 'envmiss.yaml'),'zia','locations','list')
```
**Expected:** EXIT=3, kind=missing_credentials. Verbatim (note the redaction defect): `missing zscaler API credentials: resolve client secret: <REDACTED:SECRET> secret reference: env ref is not set: ZS_NOT_SET`. The fixable part (`ZS_NOT_SET`) IS present.
**Failure means:** missing/redacted variable name → colleague can't know which var to set.
**Report:** verbatim message; confirm `ZS_NOT_SET` appears; confirm `<REDACTED:SECRET>` lands where "invalid" should be (the defect).

### D3 — file: happy (owner-only resolves)
```powershell
New-OwnerOnlyFile (Join-Path $WORK 'secret.txt') 'topsecretvalue'
$sp = (Join-Path $WORK 'secret.txt')
New-OwnerOnlyFile (Join-Path $WORK 'file.yaml') "profiles:`n  default:`n    auth_mode: oneapi`n    vanity_domain: example`n    client_id: cid`n    client_secret_ref: file:$sp"
Run-Z @('--config',(Join-Path $WORK 'file.yaml'),'zia','locations','list')
```
**Expected:** EXIT=5 (secret resolved; only tenant call failed). config show would report `set (file)`.
**Failure means:** exit 3 with `unsafe file permissions` → a correctly owner-only file is rejected → HIGH adoption blocker (owner-only unachievable with normal tooling). Capture the rejecting SID.
**Report:** EXIT; if rejected, exact text + SID; is `icacls /inheritance:r + /grant current user` sufficient to pass?

### D4 — file: broad DACL — explicit Users grant AND inherited Users
```powershell
# (a) explicit Users grant
Set-Content (Join-Path $WORK 'loose.txt') 'topsecretvalue' -NoNewline -Encoding utf8
icacls (Join-Path $WORK 'loose.txt') /grant 'Users:(R)' | Out-Null
$lp = (Join-Path $WORK 'loose.txt')
New-OwnerOnlyFile (Join-Path $WORK 'fileloose.yaml') "profiles:`n  default:`n    auth_mode: oneapi`n    vanity_domain: example`n    client_id: cid`n    client_secret_ref: file:$lp"
Show-Acl $lp
Run-Z @('--config',(Join-Path $WORK 'fileloose.yaml'),'zia','locations','list')
# (b) inherited Users (the common real case)
$ib = Join-Path $WORK 'inh'; Remove-Item $ib -Recurse -Force -ErrorAction SilentlyContinue; New-Item -ItemType Directory -Force $ib | Out-Null
icacls $ib /grant 'BUILTIN\Users:(OI)(CI)(R)' | Out-Null
Set-Content (Join-Path $ib 'sec.txt') 'topsecretvalue' -NoNewline -Encoding utf8
$ip = (Join-Path $ib 'sec.txt')
New-OwnerOnlyFile (Join-Path $WORK 'fileinh.yaml') "profiles:`n  default:`n    auth_mode: oneapi`n    vanity_domain: example`n    client_id: cid`n    client_secret_ref: file:$ip"
Show-Acl $ip
Run-Z @('--config',(Join-Path $WORK 'fileinh.yaml'),'zia','locations','list')
```
**Expected:** both EXIT=3, kind=missing_credentials. Shape: `missing zscaler API credentials: resolve client secret: <REDACTED:SECRET> credential file permissions: unsafe file permissions: broad Windows principal S-1-5-32-545 has read/write access` (explicit) and `... inherited non-owner principal S-1-5-32-545 ...` (inherited). The leading word ("unsafe") is eaten by the marker.
**Failure means:** if the message doesn't name the principal or suggest `icacls`, colleagues can't self-fix → maintainer becomes the help desk.
**Report:** verbatim reject text for (a) and (b); does it tell the user HOW to fix (it currently does NOT suggest icacls — flag); confirm no secret content leaked.

### D5 — file: missing (is the path discoverable?)
```powershell
$np = (Join-Path $WORK 'nope.txt')
New-OwnerOnlyFile (Join-Path $WORK 'filemiss.yaml') "profiles:`n  default:`n    auth_mode: oneapi`n    vanity_domain: example`n    client_id: cid`n    client_secret_ref: file:$np"
Run-Z @('--config',(Join-Path $WORK 'filemiss.yaml'),'zia','locations','list')
```
**Expected:** EXIT=3, `... resolve client secret: <REDACTED:SECRET> credential file: The system cannot find the file specified.` (Windows os.Open text). The eaten word is "open". **The PATH is absent from the message.**
**Failure means:** colleague can't tell which file is missing (path is invisible in both the error and config show).
**Report:** verbatim message + Windows-specific OS string; confirm the path is absent (UX gap).

### D6 — cmd: happy, no-shell, CRLF-safe
```powershell
New-OwnerOnlyFile (Join-Path $WORK 'cmdok.yaml') @"
profiles:
  default:
    auth_mode: oneapi
    vanity_domain: example
    client_id: cid
    client_secret_ref:
      cmd:
        argv: ["powershell.exe", "-NoProfile", "-Command", "Write-Output 'resolved-1234'"]
        timeout: 8s
"@
Run-Z @('--config',(Join-Path $WORK 'cmdok.yaml'),'zia','locations','list')
# cmd.exe variant
New-OwnerOnlyFile (Join-Path $WORK 'cmdok2.yaml') @"
profiles:
  default:
    auth_mode: oneapi
    vanity_domain: example
    client_id: cid
    client_secret_ref:
      cmd:
        argv: ["cmd.exe", "/c", "echo resolved-1234"]
"@
Run-Z @('--config',(Join-Path $WORK 'cmdok2.yaml'),'zia','locations','list')
```
**Expected:** both EXIT=5 — direct argv exec, no shell. Trailing CRLF trimmed (`TrimRight "\r\n"`), so no secret corruption.
**Failure means:** exit 3 → direct-exec broken (argv[0] PATH resolution) or CRLF survives (would later cause a confusing OAuth 401).
**Report:** EXIT each; confirm single-line secret (no CRLF corruption).

### D7 — cmd: kill-switch ZSCALERCTL_DISALLOW_CMD
```powershell
$env:ZSCALERCTL_DISALLOW_CMD = 'true'; Run-Z @('--config',(Join-Path $WORK 'cmdok.yaml'),'zia','locations','list')
$env:ZSCALERCTL_DISALLOW_CMD = '1';    Run-Z @('--config',(Join-Path $WORK 'cmdok.yaml'),'zia','locations','list')
$env:ZSCALERCTL_DISALLOW_CMD = 'yes';  Run-Z @('--config',(Join-Path $WORK 'cmdok.yaml'),'config','show')
Remove-Item Env:ZSCALERCTL_DISALLOW_CMD
```
**Expected:** `true` and `1` → EXIT=3 `... resolve client secret: <REDACTED:SECRET> secret reference: cmd refs are disabled`. `yes` (unparseable) → EXIT=2 invalid_config `parse ZSCALERCTL_DISALLOW_CMD` (caught eagerly at config show, `internal/config/load.go:34`).
**Failure means:** if `true`/`1` doesn't disable cmd, the fleet-wide kill-switch is ineffective. If `yes` is silently ignored, false sense of lockdown.
**Report:** EXIT for true/1/yes; confirm kill-switch message; confirm bad values are rejected not ignored.

### D8 — cmd: value-free failure (no secret/stderr leak)
```powershell
New-OwnerOnlyFile (Join-Path $WORK 'cmdfail.yaml') @"
profiles:
  default:
    auth_mode: oneapi
    vanity_domain: example
    client_id: cid
    client_secret_ref:
      cmd:
        argv: ["powershell.exe", "-NoProfile", "-Command", "[Console]::Error.WriteLine('SUPER-SECRET-LEAK'); exit 7"]
"@
Run-Z @('--config',(Join-Path $WORK 'cmdfail.yaml'),'zia','locations','list')
```
**Expected:** EXIT=3, `... resolve client secret: <REDACTED:SECRET> secret reference: cmd provider "powershell.exe" failed: stderr omitted (NN bytes)`. `SUPER-SECRET-LEAK` MUST NOT appear — only a byte count. Provider name IS shown.
**Failure means:** if `SUPER-SECRET-LEAK` appears anywhere, the value-free-failure guarantee is broken — a real leak regression. Most security-critical cmd scenario.
**Report:** confirm `SUPER-SECRET-LEAK` ABSENT, only `stderr omitted (N bytes)`; confirm provider name present.

### D9 — cmd: missing binary / empty output
```powershell
New-OwnerOnlyFile (Join-Path $WORK 'cmdmiss.yaml') @"
profiles:
  default:
    auth_mode: oneapi
    vanity_domain: example
    client_id: cid
    client_secret_ref:
      cmd:
        argv: ["C:\\nope\\helper.exe", "x"]
"@
Run-Z @('--config',(Join-Path $WORK 'cmdmiss.yaml'),'zia','locations','list')
New-OwnerOnlyFile (Join-Path $WORK 'cmdempty.yaml') @"
profiles:
  default:
    auth_mode: oneapi
    vanity_domain: example
    client_id: cid
    client_secret_ref:
      cmd:
        argv: ["powershell.exe", "-NoProfile", "-Command", "exit 0"]
"@
Run-Z @('--config',(Join-Path $WORK 'cmdempty.yaml'),'zia','locations','list')
```
**Expected:** missing → EXIT=3 `... cmd provider "C:\nope\helper.exe" failed: no stderr`. Empty → EXIT=3 `... cmd provider "powershell.exe" produced no output`. Both name the provider.
**Failure means:** missing provider name → can't tell which helper to fix. `failed: no stderr` for a missing binary is opaque (no "executable not found" hint).
**Report:** verbatim both; is `no stderr` clear enough for a non-expert to realize argv[0] doesn't exist?

### D10 — keyring: light happy + missing-entry (gold-standard message)
```powershell
cmdkey /generic:zscalerctl/test-secret /user:zscalerctl/test-secret /pass:keyring-val-1234 | Out-Null
New-OwnerOnlyFile (Join-Path $WORK 'kr.yaml') "profiles:`n  default:`n    auth_mode: oneapi`n    vanity_domain: example`n    client_id: cid`n    client_secret_ref: keyring:zscalerctl/test-secret"
Run-Z @('--config',(Join-Path $WORK 'kr.yaml'),'zia','locations','list')
cmdkey /delete:zscalerctl/test-secret | Out-Null
Run-Z @('--config',(Join-Path $WORK 'kr.yaml'),'zia','locations','list')
```
**Expected:** stored → EXIT=5 (resolved). After delete → EXIT=3 `... resolve client secret: <REDACTED:SECRET> secret reference: keyring has no entry for service="zscalerctl" key="test-secret"; store it or use env:/file:/cmd refs`. This is the BEST message — names service+key AND suggests the fix.
**Failure means:** stored entry failing to resolve → cgo-free keyring regressed on this host.
**Report:** both EXITs; verbatim missing-entry message; confirm service/key correct and the "store it" hint renders.

### D11 — config errors (eager, exit 2)
```powershell
Run-Z @('--config',(Join-Path $WORK 'ghost.yaml'),'config','show')
New-OwnerOnlyFile (Join-Path $WORK 'bad.yaml') "profiles:`n  default:`n    auth_mode: [oops"
Run-Z @('--config',(Join-Path $WORK 'bad.yaml'),'config','show')
New-OwnerOnlyFile (Join-Path $WORK 'unk.yaml') "profiles:`n  default:`n    auth_modeX: oneapi"
Run-Z @('--config',(Join-Path $WORK 'unk.yaml'),'config','show')
New-OwnerOnlyFile (Join-Path $WORK 'wp.yaml') "profiles:`n  work:`n    auth_mode: oneapi"
Run-Z @('--config',(Join-Path $WORK 'wp.yaml'),'--profile','nope','config','show')
```
**Expected:** all EXIT=2 invalid_config. not-found `config file not found`; bad-YAML `parse config file: yaml: line N: ...`; unknown-field `parse config file: yaml: unmarshal errors:\n  line N: field auth_modeX not found in type config.profileData`; wrong-profile `profile "nope" not found`.
**Failure means:** unknown-field leaks the Go type `config.profileData` (too internal for a colleague — clarity nit). If config-not-found fired for a DEFAULT (no `--config`) path it would be wrong (must be silent env-only).
**Report:** verbatim all four; flag the `config.profileData` leak; confirm wrong-profile names the profile.

### D12 — Windows default %APPDATA% path honored + missing-default silent
```powershell
$cfgdir = Join-Path $env:APPDATA 'zscalerctl'; New-Item -ItemType Directory -Force $cfgdir | Out-Null
Remove-Item (Join-Path $cfgdir 'config.yaml') -ErrorAction SilentlyContinue
Run-Z @('config','show','--format','json')   # missing default → silent env-only
New-OwnerOnlyFile (Join-Path $cfgdir 'config.yaml') "profiles:`n  default:`n    auth_mode: oneapi`n    vanity_domain: from-appdata"
Run-Z @('config','show','--format','json')
Remove-Item (Join-Path $cfgdir 'config.yaml')
```
**Expected:** missing default → EXIT=0, Config source=environment (no error). With file → EXIT=0, profile=default, vanity from-appdata, Config=config file.
**Failure means:** if a missing default ERRORS, every unconfigured colleague gets a spurious failure. If the default is at the wrong path (HOME/.config not on Windows), "drop a config in the obvious place" silently does nothing.
**Report:** confirm default = `%APPDATA%\zscalerctl\config.yaml`; confirm missing-default silent; note `%APPDATA%` is Roaming (may sync) and that a Notepad-created default still needs owner-only (icacls) or hits A5-style rejects.

---

## Report-back template (fill one per agent, per scenario)

```
HOST BASELINE
  OS (cmd /c ver):
  DomainJoined:
  APPDATA / under OneDrive:
  USERPROFILE drive DisplayRoot (redirected?):
  Defender RealTimeProtection / TamperProtected:
  ExecutionPolicy (Machine/User/CurrentUser):
  net use / net share (relevant lines):
  Agent: [Claude | Codex]   Date:

PER SCENARIO  (A1..A9, B1..B10, C1..C11, D1..D12)
  ID:
  ACCEPT / REJECT / SKIP:
  EXIT code (observed):
  EXIT code (expected): [from runbook]
  MATCH? yes/no
  icacls dump (Surface A/B):
  OWNER:
  Exact message (verbatim, stderr):
  Reject SID -> mapped principal (from icacls):
  Actionability (could a non-expert self-fix from this message alone? Y/N):
  Secret leak check (D8: SUPER-SECRET-LEAK absent? Y/N):
  Notes / surprises:

KEY VERDICTS (call these out explicitly)
  A1 GO/NO-GO (default %APPDATA% Set-Content ACCEPT?):
  A4 DAV-29 (temp-dir artifact vs real %APPDATA% bug):
  A9 remediation one-liner works (icacls /inheritance:r /grant:r %USERNAME%:F)?:
  B3 UNC FALSE-ACCEPT despite Everyone:Read?:
  B5 APPDATA under OneDrive + accepted (cloud egress)?:
  B8 FAT accept vs NULL-DACL reject:
  C5/C6/C7 SmartScreen/Defender/AppLocker block on managed box?:
  D8 value-free failure intact (no leak)?:
  Over-redaction defect observed in D2/D4/D5/D8/D10 (<REDACTED:SECRET> in help text)?:
```

---

## Coverage gaps on a single dev host (be honest with the maintainer)
- **AppLocker / WDAC (C5):** a single non-managed dev host almost certainly lacks the enterprise's application-control policy. A clean run here does NOT prove a managed finance box won't deny-execute. FAITHFUL only on a representative managed machine.
- **SmartScreen App Reputation (C6):** unsigned, low-prevalence binaries get reputation-gated mainly via Explorer/SmartScreen on managed boxes with cloud-delivered protection at stricter tiers. A console run on a dev host under-tests this.
- **Defender ML/heuristic reputation (C7):** enterprise ASR rules / stricter cloud-block levels can quarantine a low-prevalence Go exe that a dev host's default Defender clears.
- **No-toolchain self-containment (C8):** the dev host has Go installed; PATH-trimming is an approximation. Only a never-had-Go machine is conclusive.
- **Real domain ACLs / GPO inheritance (A7, A1):** without the actual finance GPO applied to the user profile, we cannot prove whether normally-created `%APPDATA%` configs inherit accept-set-clean DACLs or get GPO-injected `Users`/`Authenticated Users`/`Domain Users`. A non-domain or differently-OU'd host may not reproduce the mass false-reject.
- **Folder redirection / network home (B-surface):** if this host's profile is NOT redirected to a UNC home, B3/B4/B7 use a loopback share — close, but real cross-forest/SMB SID-mapping and share-vs-NTFS divergence on the production file server can differ.
- **OneDrive Known-Folder redirection (B5):** only reproducible if this host actually has OneDrive KFM enabled for `%APPDATA%`/Documents — otherwise the cloud-sync egress path is inferred, not observed.
- **Elevation-dependent steps (B2/B8, C5, A8):** loopback share, FAT VHD, Program Files copy, and setowner need admin. On a locked host these may be denied — those scenarios then go untested (report as skipped, do not improvise).
- **Real OAuth/tenant path:** all "secret resolved" verdicts stop at exit 5 (live_access_failed) with no real tenant; we never prove a resolved secret actually authenticates (e.g. a surviving CRLF would only bite against a live endpoint).

---

## SURFACE E — ZFS-over-NFS network volume (the maintainer's ACTUAL network storage)

> The maintainer has **no SMB/NTFS share** but **does** have ZFS exported over NFS, mounted on Windows via "Client for NFS." **Run this surface in place of Surface B's SMB-dependent scenarios (B2/B3/B7) when no SMB share exists.** NFS has no NTFS ACLs/SIDs; Windows synthesizes a security descriptor from NFS mode bits / identity mapping — typically Unix-UID SIDs in the `S-1-5-88-*` namespace (or anonymous). This is the real-world network-volume test and the strongest driver of the local-fixed-volume decision. Uses the helpers from §0.2.

### E0 — Identify the NFS mount + mapping
```powershell
$NFS = 'Z:\'   # <-- set to your ZFS/NFS mount root (drive letter or UNC)
"DriveType(NFS root) = $(DriveType (Split-Path $NFS -Qualifier))"   # EXPECT 4 = REMOTE
mount                                                                # Client-for-NFS: shows mounts + UID/GID mapping
Get-PSDrive | ? { $_.DisplayRoot -like '\\*' } | ft Name, DisplayRoot
```
**Report:** the DriveType value and the `mount` UID/GID mapping line.

### E1 — Config on the NFS mount (current behavior + SID capture) [GO/NO-GO for network homes]
**Intent:** exactly what the owner-only validator does with an NFS-backed config — the failure a colleague on a redirected network home would hit.
```powershell
$ncfg = Join-Path $NFS 'zsctl\config.yaml'
New-Item -ItemType Directory -Force (Split-Path $ncfg) | Out-Null
Set-Content -Path $ncfg -Value $BODY -Encoding utf8
Show-Acl $ncfg     # capture the OWNER SID — expect S-1-5-88-* (Unix uid) or anonymous, NOT your token SID
Run-Z @('--config', $ncfg, '--format','table','config','show')
```
**Expected (today, no volume rule):** almost certainly **REJECT** — owner SID ≠ your local token → "non-owner principal `S-1-5-88-…` has read/write", OR an `other`-readable mode maps to World → "broad principal".
**Failure means:** if it **ACCEPTs**, that's the *worse* outcome — the validator is trusting NFS-synthesized perms it cannot actually enforce (false-accept; the file may be world-readable on the ZFS side). Either way ⇒ NFS configs must be rejected by a **clear volume rule**, not a cryptic SID error.
**Report:** GetDriveType, owner SID, exact accept/reject + the full message verbatim (is it comprehensible to a non-expert?).

### E2 — file: secret on the NFS mount (lazy gate, exit 3)
**Intent:** same probe for a `file:` secret ref (resolves at run time → exit 3 `missing_credentials` today).
```powershell
$nsec = Join-Path $NFS 'zsctl\secret.txt'
Set-Content -Path $nsec -Value 'dummy-secret-value' -Encoding utf8
Show-Acl $nsec
# Point a profile's client_secret_ref at "file:$nsec" and run a command that RESOLVES
# (e.g. a zia/zpa read -> exit 5 live_access_failed if the secret resolved, exit 3 if the file was rejected)
```
**Report:** exit code, message, accept/reject.

### E3 — Confirm GetDriveType reliably discriminates NFS for the proposed rule
```powershell
"NFS = $(DriveType (Split-Path $NFS -Qualifier))   (expect 4 REMOTE)"
"C:  = $(DriveType 'C:\')                            (expect 3 FIXED)"
```
**Failure means:** if NFS does NOT report 4/REMOTE (some Client-for-NFS configs surface oddly), the local-volume rule needs `PathIsUNCW` + a filesystem-type check (NTFS) too, not GetDriveType alone. **Report** both values.

---

## SURFACE F — Desktop-control / GUI track (Opus + GPT-5.5 with desktop control)

> Shell tests can't see the dialogs a colleague hits FIRST. Run these with desktop control; **screenshot every dialog** and attach. Highest-friction, least-shell-observable steps.

### F1 — Double-click the downloaded `.exe` → SmartScreen [biggest distribution friction]
**Intent:** the literal first-run for a colleague who downloads `zscalerctl.exe`.
**Steps:** give the exe Mark-of-the-Web (download via a browser, or `Add-Content -Path .\zscalerctl.exe -Stream Zone.Identifier -Value "[ZoneTransfer]`nZoneId=3"`), then **double-click it in Explorer**.
**Observe + screenshot:** Does **"Windows protected your PC" (SmartScreen)** appear? Is there a **More info → Run anyway**, and can a **non-admin** use it? Does Defender/AppLocker hard-block instead?
**Failure means:** a hard block, or no non-admin "Run anyway," ⇒ the release needs **Authenticode/EV code-signing** before distribution (the single biggest distribution unknown; the dev host can't answer it faithfully — flag for a managed-box run).

### F2 — Create config via Explorer + Notepad (real inherited ACL, GUI path)
**Intent:** the genuine colleague workflow vs shell-forged ACLs. In Explorer: New → Text Document, edit in Notepad, **Save As** `config.yaml` (Save as type: **All Files**) in `%APPDATA%\zscalerctl\`. Then `Run-Z @('config','show')`.
**Report + screenshot:** did it save as `config.yaml` (not `.txt`)? Accept/reject? Cross-check against shell-created A1/A2.

### F3 — Map the ZFS/NFS export via the GUI (Map Network Drive)
**Intent:** how a colleague actually mounts network storage. Explorer → Map Network Drive (or "Connect using different credentials"), place a config there, run `config show`. Confirm the (expected) reject reads comprehensibly to a GUI user.
**Report + screenshot.**

### F4 — Application-control prompts (managed-policy dependent)
**Intent:** capture any AppLocker/WDAC/Defender **block dialog** the policy raises on the unsigned exe, run from Downloads, from `%LOCALAPPDATA%\Programs`, and from a network path.
**Report + screenshot.** If nothing triggers on THIS host, note it — only a representative managed box is conclusive (see coverage gaps).

---

> **Running order:** the **shell agent** runs Surfaces A / C / D / E. The **desktop-control agent** adds Surface F (and can cross-run E via the GUI). Surface E **replaces** B2/B3/B7 when no SMB share exists; run B1/B5/B6/B8 (local / OneDrive / removable / FAT) as written if those volumes are available.
