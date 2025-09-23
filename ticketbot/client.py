"""HTTP client for interacting with the KHAM ticketing website."""

from __future__ import annotations

import logging
import time
from dataclasses import dataclass
from http.cookiejar import CookieJar
from typing import Dict, Iterable, Optional, Tuple
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urljoin
from urllib.request import HTTPCookieProcessor, OpenerDirector, Request, build_opener

from .form_parser import FormDetails, FormParser


_LOGGER = logging.getLogger(__name__)


@dataclass
class Page:
    url: str
    status: int
    body: str


@dataclass
class LoginResult:
    success: bool
    message: str
    page: Optional[Page] = None


class KhamTicketClient:
    """A minimal HTTP client tailored for the KHAM site."""

    def __init__(self, base_url: str, *, timeout: float = 15.0, user_agent: Optional[str] = None) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.cookie_jar = CookieJar()
        self._opener: OpenerDirector = build_opener(HTTPCookieProcessor(self.cookie_jar))
        self._user_agent = user_agent or (
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
            "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"
        )
        self.last_page: Optional[Page] = None

    def _create_request(self, url: str, data: Optional[Dict[str, str]] = None, method: Optional[str] = None) -> Request:
        if not url.startswith("http"):
            url = urljoin(self.base_url + "/", url)
        encoded_data = None
        if data is not None:
            encoded_data = urlencode(data).encode("utf-8")
        request = Request(url, data=encoded_data)
        if method:
            request.get_method = lambda m=method.upper(): m  # type: ignore[assignment]
        request.add_header("User-Agent", self._user_agent)
        request.add_header("Origin", self.base_url)
        request.add_header("Referer", self.base_url)
        request.add_header("Content-Type", "application/x-www-form-urlencoded")
        return request

    def _open(self, request: Request) -> Page:
        response = self._opener.open(request, timeout=self.timeout)
        raw = response.read()
        charset = "utf-8"
        if response.headers.get_content_charset():
            charset = response.headers.get_content_charset() or charset
        text = raw.decode(charset, errors="ignore")
        page = Page(url=response.geturl(), status=getattr(response, "status", response.getcode()), body=text)
        self.last_page = page
        return page

    def fetch(self, url: str) -> Page:
        request = self._create_request(url)
        return self._open(request)

    def submit(self, form: FormDetails, overrides: Optional[Dict[str, str]] = None) -> Page:
        action, method, payload = form.merged_with(overrides)
        request = self._create_request(action, payload, method)
        return self._open(request)

    def login(
        self,
        account: str,
        password: str,
        *,
        login_page: Optional[str] = None,
        extra_overrides: Optional[Dict[str, str]] = None,
    ) -> LoginResult:
        login_url = login_page or self.base_url
        try:
            page = self.fetch(login_url)
        except (URLError, HTTPError) as exc:
            return LoginResult(success=False, message=f"Unable to load login page: {exc}")

        parser = FormParser(login_url)
        form = parser.find_first(page.body, include_password=True)
        if form is None:
            return LoginResult(success=False, message="Login form not found", page=page)

        username_field, password_field = self._resolve_login_fields(form)
        if username_field is None or password_field is None:
            return LoginResult(success=False, message="Unable to identify login fields", page=page)

        overrides = {username_field: account, password_field: password}
        if extra_overrides:
            overrides.update(extra_overrides)

        try:
            result_page = self.submit(form, overrides)
        except (URLError, HTTPError) as exc:
            return LoginResult(success=False, message=f"Login request failed: {exc}", page=page)

        if self._detect_login_failure(result_page.body):
            return LoginResult(success=False, message="Login rejected by server", page=result_page)
        return LoginResult(success=True, message="Login succeeded", page=result_page)

    def _resolve_login_fields(self, form: FormDetails) -> Tuple[Optional[str], Optional[str]]:
        password_field = None
        for name, field_type in form.field_types.items():
            if field_type == "password":
                password_field = name
                break
        username_field = None
        priorities = ("account", "member", "userid", "username", "login", "email", "id")
        for name, field_type in form.field_types.items():
            if name == password_field:
                continue
            if field_type in {"text", "email", "tel"}:
                lowered = name.lower()
                for keyword in priorities:
                    if keyword in lowered:
                        username_field = name
                        break
                if username_field:
                    break
                if username_field is None:
                    username_field = name
        return username_field, password_field

    @staticmethod
    def _detect_login_failure(body: str) -> bool:
        failure_signatures: Iterable[str] = (
            "登入失敗",
            "login failed",
            "密碼錯誤",
            "驗證碼",
        )
        lowered = body.lower()
        return any(signature in lowered for signature in failure_signatures)

    def poll_until(
        self,
        url: str,
        *,
        predicate,
        interval: float = 0.5,
        max_attempts: int = 30,
    ) -> Optional[Page]:
        for attempt in range(max_attempts):
            page = self.fetch(url)
            if predicate(page):
                return page
            time.sleep(interval)
        return None

