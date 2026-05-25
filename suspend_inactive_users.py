"""
Suspend Google Workspace users inactive > N days.

Required env vars:
  GOOGLE_APPLICATION_CREDENTIALS   JSON string of GCP service account key (domain-wide delegation)
  WORKSPACE_ADMIN_EMAIL            Admin email to impersonate
  WORKSPACE_CUSTOMER_ID            Google Workspace customer ID

Optional env vars:
  SUSPENDED_USERS_MAX_AGE_DAYS     Days since last login threshold (default: 90)
  EXCLUDE_OU                       Comma-separated OU paths to skip (default: /excluded-ou)
  DRY_RUN                          "true" = log only, "false" = suspend (default: true)
"""

import csv
import io
import json
import os
import sys
from datetime import datetime, timedelta, timezone

from google.oauth2 import service_account
from googleapiclient.discovery import build
from googleapiclient.errors import HttpError

SCOPES = ["https://www.googleapis.com/auth/admin.directory.user"]


def log(level: str, msg: str) -> None:
    print(f"[{level}] {msg}")


def get_env_str(key: str, default: str | None = None) -> str:
    val = os.getenv(key, default)
    if val is None:
        log("ERROR", f"required env var {key} not set")
        sys.exit(1)
    return val


def get_env_int(key: str, default: int) -> int:
    raw = os.getenv(key)
    if raw is None:
        return default
    try:
        return int(raw)
    except ValueError:
        log("ERROR", f"env var {key} must be integer, got '{raw}'")
        sys.exit(1)


def get_env_bool(key: str, default: bool) -> bool:
    raw = os.getenv(key)
    if raw is None:
        return default
    return raw.strip().lower() == "true"


def build_service() -> any:
    sa_json = get_env_str("GOOGLE_APPLICATION_CREDENTIALS")
    admin_email = get_env_str("WORKSPACE_ADMIN_EMAIL")

    try:
        sa_info = json.loads(sa_json)
    except json.JSONDecodeError as e:
        log("ERROR", "GOOGLE_APPLICATION_CREDENTIALS is not valid JSON: " + str(e))
        sys.exit(1)

    try:
        creds = service_account.Credentials.from_service_account_info(
            sa_info, scopes=SCOPES, subject=admin_email
        )
        return build("admin", "directory_v1", credentials=creds)
    except Exception as e:
        log("ERROR", f"failed to build service client: {e}")
        sys.exit(1)


def list_candidates(service: any, customer: str, exclude_ous: set[str], cutoff: datetime) -> list[dict]:
    candidates: list[dict] = []
    page_token: str | None = None

    while True:
        req = service.users().list(
            customer=customer,
            projection="basic",
            viewType="admin",
            maxResults=500,
            pageToken=page_token,
        )
        try:
            resp = req.execute()
        except HttpError as e:
            log("ERROR", f"API error listing users: {e}")
            sys.exit(1)

        for user in resp.get("users", []):
            email = user.get("primaryEmail", "?")
            ou = user.get("orgUnitPath", "/")
            suspended = user.get("suspended", False)

            if suspended:
                continue
            if ou in exclude_ous:
                continue

            last_login_raw = user.get("lastLoginTime")
            if not last_login_raw:
                # never logged in — skip to be safe
                continue

            try:
                last_login = datetime.fromisoformat(
                    last_login_raw.replace("Z", "+00:00")
                )
            except ValueError:
                log("WARN", f"unparseable lastLoginTime '{last_login_raw}' for {email}, skipping")
                continue

            if last_login < cutoff:
                candidates.append(
                    {"email": email, "last_login": last_login_raw, "ou": ou}
                )

        page_token = resp.get("nextPageToken")
        if not page_token:
            break

    return candidates


def suspend_user(service: any, email: str) -> bool:
    try:
        service.users().update(
            userKey=email, body={"suspended": True}
        ).execute()
        return True
    except HttpError as e:
        log("ERROR", f"suspending {email}: {e}")
        return False


def print_csv_report(candidates: list[dict]) -> None:
    buf = io.StringIO()
    w = csv.writer(buf)
    w.writerow(["email", "last_login", "ou"])
    for c in candidates:
        w.writerow([c["email"], c["last_login"], c["ou"]])
    print(buf.getvalue().rstrip())


def main() -> None:
    customer = get_env_str("WORKSPACE_CUSTOMER_ID")
    max_age_days = get_env_int("SUSPENDED_USERS_MAX_AGE_DAYS", 90)
    exclude_ous = {
        ou.strip()
        for ou in os.getenv("EXCLUDE_OU", "/excluded-ou").split(",")
        if ou.strip()
    }
    dry_run = get_env_bool("DRY_RUN", True)

    cutoff = datetime.now(timezone.utc) - timedelta(days=max_age_days)
    log("INFO", f"Target: suspend users with lastLogin < {cutoff.isoformat()}")
    log("INFO", f"Config: maxAgeDays={max_age_days}, excludeOUs={exclude_ous}, dryRun={dry_run}")

    service = build_service()
    candidates = list_candidates(service, customer, exclude_ous, cutoff)

    log("INFO", f"Found {len(candidates)} candidate(s)")

    succeeded = 0
    failed = 0
    for c in candidates:
        email = c["email"]
        last = c["last_login"]
        ou = c["ou"]

        if dry_run:
            log("INFO", f"WOULD SUSPEND {email} | lastLogin={last} | ou={ou}")
            succeeded += 1
        else:
            ok = suspend_user(service, email)
            if ok:
                log("INFO", f"SUSPENDED {email} | lastLogin={last} | ou={ou}")
                succeeded += 1
            else:
                log("ERROR", f"FAILED {email}")
                failed += 1

    print()
    log("INFO", json.dumps({
        "checked": len(candidates),
        "suspended": succeeded,
        "failed": failed,
        "dry_run": dry_run,
    }, indent=2))

    log("INFO", "CSV report:")
    print_csv_report(candidates)

    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
