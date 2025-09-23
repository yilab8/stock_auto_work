"""KHAM ticket bot package."""

from .client import KhamTicketClient, LoginResult, Page
from .form_parser import FormDetails, FormParser

__all__ = [
    "FormDetails",
    "FormParser",
    "KhamTicketClient",
    "LoginResult",
    "Page",
]
