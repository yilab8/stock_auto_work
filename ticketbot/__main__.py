"""Entry-point module for running the ticket bot as ``python -m ticketbot``."""

from __future__ import annotations

from .cli import main


def _run() -> int:
    """Invoke the CLI main function."""

    return main()


if __name__ == "__main__":  # pragma: no cover - manual invocation
    raise SystemExit(_run())
