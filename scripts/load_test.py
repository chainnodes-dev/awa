#!/usr/bin/env python3
"""
ASM Platform — API load-test script
====================================
Creates N workflow runs against the ASM Platform API and reports results.

Usage examples
--------------
  # Basic — uses .env defaults, 100 runs, 10 parallel workers
  python scripts/load_test.py

  # Override everything on the command line
  python scripts/load_test.py \\
      --url http://localhost:8080 \\
      --username admin --password secret \\
      --workflow api-test --version latest \\
      --count 100 --concurrency 10

  # Dry-run: resolve the workflow & print one sample payload, then exit
  python scripts/load_test.py --dry-run

Dependencies
------------
  pip install requests python-dotenv
  (tqdm is optional — progress bar shows automatically when installed)

Authentication
--------------
  The script logs in once to obtain a JWT access token and a refresh token.
  It auto-refreshes the access token 30 s before it would expire, so long
  runs never fail with 401s.  A single shared session is used across all
  worker threads (requests.Session is thread-safe for sending).
"""

from __future__ import annotations

import argparse
import json
import os
import random
import string
import sys
import threading
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any

# ── optional deps ──────────────────────────────────────────────────────────────
try:
    from dotenv import load_dotenv
    load_dotenv()          # load .env from cwd if present
except ImportError:
    pass                   # python-dotenv not installed — fine

try:
    import requests
except ImportError:
    sys.exit("ERROR: 'requests' package not installed.\n"
             "Run: pip install requests python-dotenv")

try:
    from tqdm import tqdm as _tqdm
    HAS_TQDM = True
except ImportError:
    HAS_TQDM = False


# ══════════════════════════════════════════════════════════════════════════════
# Configuration
# ══════════════════════════════════════════════════════════════════════════════

def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(
        description="ASM Platform API load-test",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    p.add_argument("--url",         default=os.getenv("ASM_URL", "http://localhost:8080"),
                   help="Base URL of the ASM Platform server")
    p.add_argument("--username",    default=os.getenv("ASM_USERNAME", "admin"),
                   help="Login username")
    p.add_argument("--password",    default=os.getenv("ASM_PASSWORD", ""),
                   help="Login password")
    p.add_argument("--tenant",      default=os.getenv("ASM_TENANT", "default"),
                   help="Tenant slug")
    p.add_argument("--workflow",    default=os.getenv("ASM_WORKFLOW", "api-test"),
                   help="Workflow name to use")
    p.add_argument("--version",     default=os.getenv("ASM_WORKFLOW_VERSION", "latest"),
                   help="Workflow version (use 'latest' to auto-detect)")
    p.add_argument("--count",       type=int, default=int(os.getenv("LOAD_COUNT", "100")),
                   help="Number of runs to create")
    p.add_argument("--concurrency", type=int, default=int(os.getenv("LOAD_CONCURRENCY", "10")),
                   help="Max parallel workers")
    p.add_argument("--delay",       type=float, default=float(os.getenv("LOAD_DELAY", "0.0")),
                   help="Extra delay (seconds) between each request — 0 = as fast as possible")
    p.add_argument("--dry-run",     action="store_true",
                   help="Resolve workflow, print a sample payload, then exit")
    p.add_argument("--no-color",    action="store_true",
                   help="Disable ANSI colour output")
    return p.parse_args()


# ══════════════════════════════════════════════════════════════════════════════
# ANSI colours
# ══════════════════════════════════════════════════════════════════════════════

class C:
    RESET  = "\033[0m"
    BOLD   = "\033[1m"
    GREEN  = "\033[32m"
    YELLOW = "\033[33m"
    RED    = "\033[31m"
    CYAN   = "\033[36m"
    GREY   = "\033[90m"

_color_enabled = True

def col(code: str, text: str) -> str:
    return f"{code}{text}{C.RESET}" if _color_enabled else text


# ══════════════════════════════════════════════════════════════════════════════
# Auth manager  (thread-safe token storage + auto-refresh)
# ══════════════════════════════════════════════════════════════════════════════

@dataclass
class TokenState:
    access_token:   str   = ""
    refresh_token:  str   = ""
    expires_at:     float = 0.0   # unix timestamp
    _lock: threading.Lock = field(default_factory=threading.Lock, repr=False)


class AuthManager:
    """
    Handles login and transparent token refresh.
    Thread-safe: multiple workers share one AuthManager instance.
    """

    REFRESH_BEFORE = 30  # seconds before expiry to proactively refresh

    def __init__(self, session: requests.Session, base_url: str,
                 username: str, password: str, tenant: str):
        self._session  = session
        self._base_url = base_url.rstrip("/")
        self._username = username
        self._password = password
        self._tenant   = tenant
        self._state    = TokenState()

    # ── public ────────────────────────────────────────────────────────────────

    def login(self) -> None:
        """Obtain initial token pair.  Raises on failure."""
        resp = self._session.post(
            f"{self._base_url}/api/v1/auth/login",
            json={
                "username":    self._username,
                "password":    self._password,
                "tenant_slug": self._tenant,
            },
            timeout=15,
        )
        self._raise_for_status(resp, "login")
        data = resp.json()
        self._store(data)
        print(col(C.GREEN, "✓") + f" Logged in as {col(C.BOLD, self._username)}"
              f"  (token expires in {data.get('expires_in', '?')}s)")

    def auth_header(self) -> dict[str, str]:
        """Return a ready-to-use Authorization header, refreshing if needed."""
        with self._state._lock:
            if time.time() >= self._state.expires_at - self.REFRESH_BEFORE:
                self._do_refresh()
            return {"Authorization": f"Bearer {self._state.access_token}"}

    # ── private ───────────────────────────────────────────────────────────────

    def _do_refresh(self) -> None:
        """Must be called with _state._lock held."""
        resp = self._session.post(
            f"{self._base_url}/api/v1/auth/refresh",
            json={"refresh_token": self._state.refresh_token},
            timeout=15,
        )
        self._raise_for_status(resp, "token refresh")
        self._store(resp.json())

    def _store(self, data: dict) -> None:
        self._state.access_token  = data["access_token"]
        self._state.refresh_token = data["refresh_token"]
        expires_in = data.get("expires_in", 900)
        self._state.expires_at = time.time() + expires_in

    @staticmethod
    def _raise_for_status(resp: requests.Response, action: str) -> None:
        if not resp.ok:
            try:
                detail = resp.json()
            except Exception:
                detail = resp.text
            raise RuntimeError(f"{action} failed ({resp.status_code}): {detail}")


# ══════════════════════════════════════════════════════════════════════════════
# Workflow resolution
# ══════════════════════════════════════════════════════════════════════════════

def _workflow_list(session: requests.Session, headers: dict, base: str) -> list[dict]:
    """
    Fetch all workflows (latest version of each) from GET /api/v1/workflows.

    The API returns []*WorkflowDef, each shaped like:
        {
          "apiVersion": "asm/v1",
          "metadata": { "name": "api-test", "version": "31", "version_number": 31, ... },
          "states": [...],
          ...
        }
    Returns a list of dicts (may be empty; never None).
    """
    resp = session.get(f"{base}/api/v1/workflows", headers=headers, timeout=10)
    if not resp.ok:
        return []
    raw = resp.json()
    return raw if isinstance(raw, list) else []


def resolve_workflow(session: requests.Session, auth: AuthManager,
                     base_url: str, name: str, version: str) -> tuple[str, str]:
    """
    Return (workflow_name, resolved_version_string).

    Strategy:
    - GET /api/v1/workflows returns one WorkflowDef per workflow (DISTINCT ON name,
      highest version_number wins).  Each entry carries metadata.name and
      metadata.version — the exact strings needed for StartRun.
    - For version == 'latest' we read the version directly from that entry.
    - For a specific version string we fall through to GET /{name}/versions to
      verify it exists.
    """
    base = base_url.rstrip("/")
    headers = auth.auth_header()

    all_workflows = _workflow_list(session, headers, base)

    # Name is nested under "metadata", not at the top level.
    def wf_name(w: dict) -> str:
        return w.get("metadata", {}).get("name", "")

    def wf_version(w: dict) -> str:
        return w.get("metadata", {}).get("version", "")

    def wf_version_number(w: dict) -> int:
        return w.get("metadata", {}).get("version_number", 0)

    # Case-insensitive match so "api-test" finds "API-test"
    match = next((w for w in all_workflows if wf_name(w).lower() == name.lower()), None)

    if version.lower() == "latest":
        if match is None:
            _die_not_found(name, all_workflows)

        resolved_ver = wf_version(match)
        resolved_num = wf_version_number(match)
        canonical_name = wf_name(match)          # use the server's casing
        if not resolved_ver:
            _die_not_found(name, all_workflows,
                           extra="found but has no version string — try saving it again in the Designer.")
        print(col(C.CYAN, "ℹ") + f"  Resolved {col(C.BOLD, canonical_name)} → "
              f"version {col(C.BOLD, resolved_ver)} (#{resolved_num})")
        return canonical_name, resolved_ver

    else:
        # Specific version requested — verify via the versions list endpoint
        canonical_name = wf_name(match) if match else name
        resp = session.get(f"{base}/api/v1/workflows/{canonical_name}/versions",
                           headers=headers, timeout=10)
        if resp.status_code == 404:
            _die_not_found(name, all_workflows)
        _raise(resp, f"list versions for {canonical_name}")

        raw = resp.json()
        summaries: list[dict] = raw if isinstance(raw, list) else []
        available = [s.get("version", "") for s in summaries]

        if version not in available:
            _die_not_found(name, all_workflows,
                           extra=f"version '{version}' not found. Available: {available}")

        print(col(C.CYAN, "ℹ") + f"  Using {col(C.BOLD, canonical_name)} version {col(C.BOLD, version)}")
        return canonical_name, version


def _die_not_found(name: str, all_workflows: list[dict], extra: str = "") -> None:
    """Print a combined error + available-workflow list, then exit."""
    msg = f"Workflow '{name}' not found on this server."
    if extra:
        msg += f"\n  ({extra})"
    print(col(C.RED, f"\n✗ {msg}"))
    if all_workflows:
        print(col(C.YELLOW, "\n  Available workflows on this server:"))
        for w in all_workflows:
            wname = w.get("metadata", {}).get("name", "?")
            wver  = w.get("metadata", {}).get("version", "?")
            wnum  = w.get("metadata", {}).get("version_number", "?")
            print(f"    • {wname}  (latest: v{wnum} / \"{wver}\")")
    else:
        print(col(C.YELLOW, "  No workflows found on this server."))
    raise SystemExit(1)


def _raise(resp: requests.Response, action: str) -> None:
    if not resp.ok:
        try:
            detail = resp.json()
        except Exception:
            detail = resp.text
        raise RuntimeError(f"{action} failed ({resp.status_code}): {detail}")


# ══════════════════════════════════════════════════════════════════════════════
# Workflow definition fetch
# ══════════════════════════════════════════════════════════════════════════════

def fetch_workflow_def(session: requests.Session, auth: AuthManager,
                       base_url: str, name: str, version: str) -> dict:
    """
    Fetch the full workflow definition (GET /api/v1/workflows/{name}/{version}).
    Returns the 'definition' sub-object which contains the blackboard schema.
    """
    resp = session.get(
        f"{base_url.rstrip('/')}/api/v1/workflows/{name}/{version}",
        headers=auth.auth_header(),
        timeout=10,
    )
    _raise(resp, f"fetch definition {name}/{version}")
    data = resp.json()
    # Response shape: {"definition": {...}, "yaml": "..."}
    return data.get("definition", data)


# ══════════════════════════════════════════════════════════════════════════════
# Test-data generator  (schema-driven)
# ══════════════════════════════════════════════════════════════════════════════

_WORDS = ["alpha", "beta", "gamma", "delta", "echo", "foxtrot", "golf",
          "hotel", "india", "juliet", "kilo", "lima", "mike", "november"]


def _rnd_id(prefix: str = "", length: int = 8) -> str:
    return prefix + "".join(random.choices(string.ascii_uppercase + string.digits, k=length))


def _sample_value(field_name: str, field_def: dict, rng: random.Random) -> Any:
    """
    Generate a plausible value for a blackboard field based on its declared type
    and name hints.  Falls back to safe defaults for unknown types.
    """
    ftype = (field_def.get("type") or "string").lower()
    default = field_def.get("default")

    # ── string ────────────────────────────────────────────────────────────────
    if ftype == "string":
        name_l = field_name.lower()
        if any(k in name_l for k in ("id", "ref", "code", "number", "num")):
            return _rnd_id("", 8)
        if any(k in name_l for k in ("email", "mail")):
            return f"user{rng.randint(1, 999)}@example.com"
        if any(k in name_l for k in ("name", "title", "label")):
            return f"{rng.choice(_WORDS).capitalize()} {rng.choice(_WORDS).capitalize()}"
        if any(k in name_l for k in ("status", "state", "type", "category", "kind")):
            return rng.choice(["pending", "active", "review", "approved"])
        if any(k in name_l for k in ("priority", "prio")):
            return rng.choice(["low", "medium", "high", "critical"])
        if any(k in name_l for k in ("desc", "note", "comment", "body", "text", "message")):
            return f"Automated test entry #{rng.randint(1000, 9999)} — {rng.choice(_WORDS)}"
        if any(k in name_l for k in ("url", "link", "uri")):
            return f"https://example.com/{rng.choice(_WORDS)}/{rng.randint(1, 999)}"
        if any(k in name_l for k in ("date", "time", "at")):
            return datetime.now(timezone.utc).isoformat()
        # generic fallback
        return f"{rng.choice(_WORDS)}_{rng.randint(1, 999)}"

    # ── number / integer ──────────────────────────────────────────────────────
    if ftype in ("number", "integer", "int", "float"):
        name_l = field_name.lower()
        if any(k in name_l for k in ("amount", "price", "cost", "value", "total")):
            return round(rng.uniform(100.0, 50_000.0), 2)
        if any(k in name_l for k in ("count", "qty", "quantity", "num")):
            return rng.randint(1, 50)
        if any(k in name_l for k in ("age", "year")):
            return rng.randint(18, 65)
        if any(k in name_l for k in ("percent", "pct", "rate")):
            return round(rng.uniform(0.0, 100.0), 1)
        return round(rng.uniform(1.0, 1000.0), 2)

    # ── bool ─────────────────────────────────────────────────────────────────
    if ftype in ("bool", "boolean"):
        # If there's an explicit default, use it; otherwise random
        if default is not None:
            return bool(default)
        return rng.choice([True, False])

    # ── object / map ─────────────────────────────────────────────────────────
    if ftype in ("object", "map", "dict"):
        return {"key": rng.choice(_WORDS), "value": rng.randint(1, 100)}

    # ── array / list ─────────────────────────────────────────────────────────
    if ftype in ("array", "list"):
        return [rng.choice(_WORDS) for _ in range(rng.randint(1, 3))]

    # ── unknown ───────────────────────────────────────────────────────────────
    return default if default is not None else f"test_{rng.randint(1, 999)}"


def make_payload(index: int, bb_schema: dict[str, dict]) -> dict[str, Any]:
    """
    Generate a varied payload that exactly matches the workflow's declared
    blackboard schema.  Every field in the schema gets a value; required fields
    are always populated.
    """
    rng = random.Random(index)   # seeded → reproducible per index
    payload: dict[str, Any] = {}

    for field_name, field_def in bb_schema.items():
        payload[field_name] = _sample_value(field_name, field_def, rng)

    return payload


def print_schema(bb_schema: dict[str, dict]) -> None:
    """Pretty-print the blackboard schema so the user can verify field coverage."""
    if not bb_schema:
        print(col(C.YELLOW, "  ⚠  Workflow has no blackboard schema — sending empty input."))
        return
    print(col(C.CYAN, "ℹ") + "  Blackboard schema:")
    for fname, fdef in bb_schema.items():
        req  = " (required)" if fdef.get("required") else ""
        typ  = fdef.get("type", "string")
        dflt = f"  default={fdef['default']}" if "default" in fdef else ""
        print(f"    • {col(C.BOLD, fname):30s}  {typ}{req}{dflt}")


# ══════════════════════════════════════════════════════════════════════════════
# Single run worker
# ══════════════════════════════════════════════════════════════════════════════

@dataclass
class RunResult:
    index:       int
    success:     bool
    run_id:      str  = ""
    status_code: int  = 0
    error:       str  = ""
    elapsed_ms:  int  = 0


def start_run(
    index:     int,
    session:   requests.Session,
    auth:      AuthManager,
    base_url:  str,
    wf_name:   str,
    wf_ver:    str,
    delay:     float,
    bb_schema: dict,
) -> RunResult:
    if delay > 0:
        time.sleep(delay)

    payload = make_payload(index, bb_schema)
    t0 = time.monotonic()
    try:
        resp = session.post(
            f"{base_url.rstrip('/')}/api/v1/runs",
            json={
                "workflow_name":    wf_name,
                "workflow_version": wf_ver,
                "input":            payload,
            },
            headers=auth.auth_header(),
            timeout=30,
        )
        elapsed = int((time.monotonic() - t0) * 1000)

        if resp.ok:
            run_id = resp.json().get("id", "unknown")
            return RunResult(index=index, success=True, run_id=run_id,
                             status_code=resp.status_code, elapsed_ms=elapsed)
        else:
            try:
                detail = str(resp.json())
            except Exception:
                detail = resp.text[:200]
            return RunResult(index=index, success=False, status_code=resp.status_code,
                             error=detail, elapsed_ms=elapsed)

    except Exception as exc:
        elapsed = int((time.monotonic() - t0) * 1000)
        return RunResult(index=index, success=False, error=str(exc), elapsed_ms=elapsed)


# ══════════════════════════════════════════════════════════════════════════════
# Summary report
# ══════════════════════════════════════════════════════════════════════════════

def print_summary(results: list[RunResult], wall_time: float) -> None:
    ok  = [r for r in results if r.success]
    err = [r for r in results if not r.success]

    latencies = [r.elapsed_ms for r in ok]
    avg_ms = int(sum(latencies) / len(latencies)) if latencies else 0
    min_ms = min(latencies) if latencies else 0
    max_ms = max(latencies) if latencies else 0
    p95_ms = sorted(latencies)[int(len(latencies) * 0.95)] if latencies else 0

    throughput = len(results) / wall_time if wall_time > 0 else 0

    print()
    print(col(C.BOLD, "═" * 56))
    print(col(C.BOLD, "  RESULTS"))
    print(col(C.BOLD, "═" * 56))
    print(f"  Total runs submitted : {len(results)}")
    print(f"  {col(C.GREEN, 'Successful')}           : {len(ok)}")
    if err:
        print(f"  {col(C.RED, 'Failed')}               : {len(err)}")
    print()
    if latencies:
        print(col(C.BOLD, "  Latency (ms)"))
        print(f"    avg : {avg_ms:>6}")
        print(f"    min : {min_ms:>6}")
        print(f"    p95 : {p95_ms:>6}")
        print(f"    max : {max_ms:>6}")
        print()
    print(f"  Wall time  : {wall_time:.2f}s")
    print(f"  Throughput : {throughput:.1f} runs/s")
    print(col(C.BOLD, "═" * 56))

    if err:
        print()
        print(col(C.YELLOW, "  Failed runs (first 10):"))
        for r in err[:10]:
            print(f"    #{r.index+1:>3}  HTTP {r.status_code or '---'}  {r.error[:80]}")

    # Error breakdown by HTTP status code
    if err:
        from collections import Counter
        codes = Counter(r.status_code for r in err)
        print()
        print(col(C.YELLOW, "  Error breakdown by status code:"))
        for code, count in sorted(codes.items()):
            print(f"    HTTP {code or '---'} : {count}x")

    print()
    if not err:
        print(col(C.GREEN, "  ✓ All runs created successfully!"))
    else:
        pct = 100 * len(ok) / len(results)
        print(col(C.YELLOW, f"  ⚠  Success rate: {pct:.1f}%"))


# ══════════════════════════════════════════════════════════════════════════════
# Main
# ══════════════════════════════════════════════════════════════════════════════

def main() -> None:
    global _color_enabled

    args = parse_args()
    _color_enabled = not args.no_color and sys.stdout.isatty()

    if not args.password:
        print(col(C.YELLOW, "⚠  No password provided."))
        print("   Set --password, the ASM_PASSWORD env var, or add it to .env")
        raise SystemExit(1)

    print()
    print(col(C.BOLD, "ASM Platform — API Load Test"))
    print(col(C.GREY,  f"  Server  : {args.url}"))
    print(col(C.GREY,  f"  User    : {args.username} @ {args.tenant}"))
    print(col(C.GREY,  f"  Target  : {args.workflow} v{args.version}"))
    print(col(C.GREY,  f"  Runs    : {args.count}  concurrency={args.concurrency}"))
    print()

    # ── shared session (connection pool) ─────────────────────────────────────
    session = requests.Session()
    session.headers.update({"Content-Type": "application/json"})

    # ── authenticate ──────────────────────────────────────────────────────────
    auth = AuthManager(session, args.url, args.username, args.password, args.tenant)
    try:
        auth.login()
    except RuntimeError as exc:
        print(col(C.RED, f"\n✗ Authentication failed: {exc}"))
        raise SystemExit(1)

    # ── resolve workflow ──────────────────────────────────────────────────────
    try:
        wf_name, wf_ver = resolve_workflow(
            session, auth, args.url, args.workflow, args.version)
    except RuntimeError as exc:
        print(col(C.RED, f"\n✗ {exc}"))
        raise SystemExit(1)

    # ── fetch definition & extract blackboard schema ──────────────────────────
    try:
        wf_def   = fetch_workflow_def(session, auth, args.url, wf_name, wf_ver)
        bb_schema: dict[str, dict] = (
            wf_def.get("blackboard", {}).get("schema", {}) or {}
        )
    except RuntimeError as exc:
        print(col(C.RED, f"\n✗ Could not fetch workflow definition: {exc}"))
        raise SystemExit(1)

    print_schema(bb_schema)

    # ── dry-run ───────────────────────────────────────────────────────────────
    if args.dry_run:
        sample = make_payload(0, bb_schema)
        print()
        print(col(C.BOLD, "  Dry-run — sample payload for run #1:"))
        print(json.dumps({
            "workflow_name":    wf_name,
            "workflow_version": wf_ver,
            "input":            sample,
        }, indent=2))
        print()
        print(col(C.GREEN, "  ✓ Dry-run complete. No runs were submitted."))
        return

    # ── submit runs ───────────────────────────────────────────────────────────
    print()
    results: list[RunResult] = []
    t_start = time.monotonic()

    if HAS_TQDM:
        pbar = _tqdm(total=args.count, desc="Submitting runs",
                     unit="run", dynamic_ncols=True)
    else:
        pbar = None
        print(f"  Submitting {args.count} runs "
              f"({args.concurrency} workers)…  ", end="", flush=True)

    with ThreadPoolExecutor(max_workers=args.concurrency) as pool:
        futures = {
            pool.submit(start_run, i, session, auth,
                        args.url, wf_name, wf_ver, args.delay, bb_schema): i
            for i in range(args.count)
        }
        completed = 0
        for future in as_completed(futures):
            result = future.result()
            results.append(result)
            completed += 1

            if pbar:
                icon = "✓" if result.success else "✗"
                pbar.set_postfix_str(
                    f"{icon} #{result.index+1} "
                    + (f"→ {result.run_id[-12:]}" if result.success
                       else f"ERR {result.status_code}")
                )
                pbar.update(1)
            else:
                # Simple inline progress every 10 runs
                if completed % 10 == 0 or completed == args.count:
                    pct = int(100 * completed / args.count)
                    print(f"{pct}%… ", end="", flush=True)

    if pbar:
        pbar.close()
    else:
        print("done")

    wall_time = time.monotonic() - t_start

    # Sort by original index before printing
    results.sort(key=lambda r: r.index)

    print_summary(results, wall_time)


if __name__ == "__main__":
    main()
