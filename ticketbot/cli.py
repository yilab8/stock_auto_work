"""Command line interface for the KHAM ticket bot."""

from __future__ import annotations

import argparse
import json
import os
import sys
from typing import Dict, Iterable, List, Optional

from .client import KhamTicketClient
from .form_parser import FormDetails, FormParser

DEFAULT_BASE_URL = "https://kham.com.tw/application/utk01/UTK0101_03.aspx"
DEFAULT_TIMEOUT = 15.0


def _add_common_arguments(parser: argparse.ArgumentParser) -> None:
    parser.add_argument(
        "--base-url",
        default=DEFAULT_BASE_URL,
        help="Base URL of the ticketing page.",
    )
    parser.add_argument(
        "--timeout",
        type=float,
        default=DEFAULT_TIMEOUT,
        help="Request timeout in seconds.",
    )


def _parse_key_value_pairs(pairs: Iterable[str]) -> Dict[str, str]:
    result: Dict[str, str] = {}
    for pair in pairs:
        if "=" not in pair:
            raise ValueError(f"Invalid key=value pair: {pair}")
        key, value = pair.split("=", 1)
        result[key] = value
    return result


def command_login(args: argparse.Namespace) -> int:
    client = KhamTicketClient(args.base_url, timeout=args.timeout)
    extras = _parse_key_value_pairs(args.extra) if args.extra else None
    result = client.login(
        account=args.account,
        password=args.password,
        login_page=args.login_page,
        extra_overrides=extras,
    )
    print(result.message)
    if result.page:
        print(f"Final URL: {result.page.url}")
        print(f"Status: {result.page.status}")
    if args.dump_html and result.page:
        print(result.page.body)
    return 0 if result.success else 1


def _describe_form(form: FormDetails, index: int) -> str:
    lines = [f"Form #{index}"]
    lines.append(f"  action: {form.action}")
    lines.append(f"  method: {form.method}")
    lines.append("  fields:")
    for name, value in form.fields.items():
        field_type = form.field_types.get(name, "")
        lines.append(f"    - {name} (type={field_type}) default='{value}'")
    return "\n".join(lines)


def command_dump_forms(args: argparse.Namespace) -> int:
    client = KhamTicketClient(args.base_url, timeout=args.timeout)
    page = client.fetch(args.url or args.base_url)
    parser = FormParser(page.url)
    forms = list(parser.parse(page.body))
    if args.include_password:
        forms = [form for form in forms if any(t == "password" for t in form.field_types.values())]
    if not forms:
        print("No forms found.")
        return 1
    for idx, form in enumerate(forms, start=1):
        print(_describe_form(form, idx))
    return 0


def _resolve_env(value: str) -> str:
    if value.startswith("${") and value.endswith("}"):
        env_key = value[2:-1]
        return os.environ.get(env_key, "")
    return value


def _select_form(forms: List[FormDetails], criteria: Dict[str, object]) -> Optional[FormDetails]:
    action_contains = criteria.get("action_contains")
    required_fields = set(criteria.get("required_fields", []) or [])
    index = criteria.get("index")
    filtered = []
    for form in forms:
        if action_contains and action_contains not in form.action:
            continue
        if required_fields and not required_fields.issubset(form.fields.keys()):
            continue
        filtered.append(form)
    if index is not None:
        try:
            return filtered[int(index)]
        except (IndexError, ValueError):
            return None
    return filtered[0] if filtered else None


def _execute_step(client: KhamTicketClient, current_page, step: Dict[str, object]):
    step_type = step.get("type", "submit")
    if step_type == "fetch":
        url = step.get("url") or client.base_url
        print(f"Fetching {url} ...")
        return client.fetch(url)

    if current_page is None or not step.get("use_last_page", True):
        url = step.get("url") or client.base_url
        print(f"Fetching {url} for form submission ...")
        current_page = client.fetch(url)
    parser = FormParser(current_page.url)
    forms = list(parser.parse(current_page.body))
    criteria = step.get("form") or {}
    form = _select_form(forms, criteria)
    if form is None:
        raise RuntimeError("No form matches the provided criteria")
    overrides = {key: _resolve_env(str(value)) for key, value in (step.get("overrides") or {}).items()}
    print(f"Submitting form to {form.action} ...")
    return client.submit(form, overrides)


def command_run_config(args: argparse.Namespace) -> int:
    with open(args.config, "r", encoding="utf-8") as handle:
        config = json.load(handle)

    client = KhamTicketClient(config.get("base_url", args.base_url), timeout=config.get("timeout", args.timeout))
    login_cfg = config.get("login")
    if login_cfg:
        account = _resolve_env(str(login_cfg.get("account", "")))
        password = _resolve_env(str(login_cfg.get("password", "")))
        extra = {k: _resolve_env(str(v)) for k, v in (login_cfg.get("extra_overrides") or {}).items()}
        result = client.login(account, password, login_page=login_cfg.get("page"), extra_overrides=extra)
        print(result.message)
        if not result.success:
            return 1
        current_page = result.page
    else:
        current_page = None

    for step in config.get("steps", []):
        current_page = _execute_step(client, current_page, step)
        print(f"Step completed. URL: {current_page.url} status={current_page.status}")
        if args.verbose:
            print(current_page.body)

    poll_cfg = config.get("polling")
    if poll_cfg:
        url = poll_cfg.get("url") or (current_page.url if current_page else client.base_url)
        keyword = poll_cfg.get("keyword")
        success = client.poll_until(
            url,
            predicate=lambda page: (keyword in page.body) if keyword else True,
            interval=float(poll_cfg.get("interval", 0.5)),
            max_attempts=int(poll_cfg.get("max_attempts", 30)),
        )
        if success:
            print(f"Polling completed. URL: {success.url}")
        else:
            print("Polling finished without satisfying condition")
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="KHAM ticket bot helper")
    subparsers = parser.add_subparsers(dest="command")

    login_parser = subparsers.add_parser("login", help="Perform a login request and report the result")
    _add_common_arguments(login_parser)
    login_parser.add_argument("--account", required=True, help="Member account")
    login_parser.add_argument("--password", required=True, help="Member password")
    login_parser.add_argument("--login-page", help="Optional login page URL")
    login_parser.add_argument(
        "--extra",
        nargs="*",
        default=[],
        help="Additional form fields specified as key=value pairs",
    )
    login_parser.add_argument("--dump-html", action="store_true", help="Dump response HTML after login")
    login_parser.set_defaults(func=command_login)

    forms_parser = subparsers.add_parser("dump-forms", help="Fetch a page and print detected forms")
    _add_common_arguments(forms_parser)
    forms_parser.add_argument("--url", help="Page URL to inspect")
    forms_parser.add_argument("--include-password", action="store_true", help="Only show forms with password fields")
    forms_parser.set_defaults(func=command_dump_forms)

    run_parser = subparsers.add_parser("run", help="Execute automation based on a JSON configuration file")
    _add_common_arguments(run_parser)
    run_parser.add_argument("--config", required=True, help="Path to configuration JSON file")
    run_parser.add_argument("--verbose", action="store_true", help="Print HTML after each step")
    run_parser.set_defaults(func=command_run_config)

    return parser


def main(argv: Optional[List[str]] = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    if not hasattr(args, "func"):
        parser.print_help()
        return 1
    try:
        return args.func(args)
    except Exception as exc:  # pragma: no cover - surfaced to CLI user
        print(f"Error: {exc}")
        return 1


if __name__ == "__main__":
    sys.exit(main())
