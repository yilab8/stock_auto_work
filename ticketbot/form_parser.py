"""Utilities to parse HTML forms and prepare submission payloads."""

from __future__ import annotations

from dataclasses import dataclass
from html.parser import HTMLParser
from typing import Dict, Iterable, List, Optional, Tuple
from urllib.parse import urljoin


@dataclass
class FormDetails:
    """Representation of a HTML form that can be submitted via HTTP."""

    action: str
    method: str
    fields: Dict[str, str]
    field_types: Dict[str, str]

    def merged_with(self, overrides: Optional[Dict[str, str]] = None) -> Tuple[str, str, Dict[str, str]]:
        data = dict(self.fields)
        if overrides:
            data.update({k: v for k, v in overrides.items() if k})
        return self.action, self.method.upper(), data


class _FormHTMLParser(HTMLParser):
    def __init__(self, base_url: str) -> None:
        super().__init__()
        self.base_url = base_url
        self.forms: List[FormDetails] = []
        self._current_action: Optional[str] = None
        self._current_method: str = "GET"
        self._current_fields: Dict[str, str] = {}
        self._current_types: Dict[str, str] = {}
        self._current_select: Optional[str] = None
        self._current_select_value: Optional[str] = None
        self._current_textarea: Optional[str] = None

    def handle_starttag(self, tag: str, attrs_list):
        attrs = dict(attrs_list)
        if tag == "form":
            self._current_action = urljoin(self.base_url, attrs.get("action", ""))
            self._current_method = attrs.get("method", "GET").upper()
            self._current_fields = {}
            self._current_types = {}
        elif tag == "input" and self._current_action is not None:
            name = attrs.get("name")
            if not name:
                return
            input_type = attrs.get("type", "text").lower()
            if input_type in {"submit", "button", "image"}:
                return
            self._current_fields[name] = attrs.get("value", "")
            self._current_types[name] = input_type
        elif tag == "textarea" and self._current_action is not None:
            name = attrs.get("name")
            if not name:
                return
            self._current_textarea = name
            self._current_fields[name] = ""
            self._current_types[name] = "textarea"
        elif tag == "select" and self._current_action is not None:
            name = attrs.get("name")
            if not name:
                return
            self._current_select = name
            self._current_select_value = None
            self._current_types[name] = "select"
        elif tag == "option" and self._current_select:
            value = attrs.get("value", "")
            if self._current_select_value is None or ("selected" in attrs):
                self._current_select_value = value

    def handle_endtag(self, tag: str):
        if tag == "form" and self._current_action is not None:
            self.forms.append(
                FormDetails(
                    action=self._current_action,
                    method=self._current_method,
                    fields=dict(self._current_fields),
                    field_types=dict(self._current_types),
                )
            )
            self._current_action = None
            self._current_fields = {}
            self._current_types = {}
            self._current_select = None
            self._current_select_value = None
            self._current_textarea = None
        elif tag == "select" and self._current_select:
            self._current_fields[self._current_select] = self._current_select_value or ""
            self._current_select = None
            self._current_select_value = None
        elif tag == "textarea":
            self._current_textarea = None

    def handle_data(self, data: str):
        if self._current_select and self._current_select_value is None:
            stripped = data.strip()
            if stripped:
                self._current_select_value = stripped
        elif self._current_textarea:
            existing = self._current_fields.get(self._current_textarea, "")
            self._current_fields[self._current_textarea] = existing + data


class FormParser:
    """Parse forms from raw HTML."""

    def __init__(self, base_url: str) -> None:
        self.base_url = base_url

    def parse(self, html: str) -> Iterable[FormDetails]:
        parser = _FormHTMLParser(self.base_url)
        parser.feed(html)
        return list(parser.forms)

    def find_first(self, html: str, *, include_password: bool = False) -> Optional[FormDetails]:
        for form in self.parse(html):
            if include_password and not self._has_password_field(form):
                continue
            return form
        return None

    @staticmethod
    def _has_password_field(form: FormDetails) -> bool:
        for name, field_type in form.field_types.items():
            if field_type == "password" or "pass" in name.lower():
                return True
        return False

