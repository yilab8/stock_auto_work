"""Graphical interface for interacting with the KHAM ticket bot."""

from __future__ import annotations

import argparse
import contextlib
import io
import threading
from typing import Callable, List, Optional

try:  # pragma: no cover - runtime availability check
    import tkinter as tk
    from tkinter import filedialog, messagebox, ttk
except ImportError as exc:  # pragma: no cover - guard for environments without Tk
    tk = None  # type: ignore[assignment]
    ttk = None  # type: ignore[assignment]
    filedialog = None  # type: ignore[assignment]
    messagebox = None  # type: ignore[assignment]
    _TK_IMPORT_ERROR = exc
else:
    _TK_IMPORT_ERROR = None

from . import cli


class TicketBotGUI:
    """Tkinter based desktop interface for triggering ticket bot commands."""

    def __init__(self, master: "Optional[tk.Misc]" = None) -> None:
        if tk is None:  # pragma: no cover - exercised only when Tk is missing
            raise RuntimeError("Tkinter is required to run the GUI") from _TK_IMPORT_ERROR

        self._root = master or tk.Tk()
        self._owns_root = master is None
        self._root.title("KHAM Ticket Bot")
        self._root.geometry("800x620")

        self._base_url = tk.StringVar(value=cli.DEFAULT_BASE_URL)
        self._timeout = tk.StringVar(value=str(cli.DEFAULT_TIMEOUT))
        self._login_account = tk.StringVar()
        self._login_password = tk.StringVar()
        self._login_page = tk.StringVar()
        self._login_extra = tk.StringVar()
        self._login_dump_html = tk.BooleanVar(value=False)

        self._forms_url = tk.StringVar()
        self._forms_include_password = tk.BooleanVar(value=False)

        self._run_config_path = tk.StringVar()
        self._run_verbose = tk.BooleanVar(value=False)

        self._build_layout()

    # ------------------------------------------------------------------
    # Widget construction helpers
    # ------------------------------------------------------------------
    def _build_layout(self) -> None:
        assert tk is not None and ttk is not None  # for type checkers

        container = ttk.Frame(self._root, padding=12)
        container.pack(fill=tk.BOTH, expand=True)

        notebook = ttk.Notebook(container)
        notebook.pack(fill=tk.BOTH, expand=True)

        notebook.add(self._build_login_tab(notebook), text="登入")
        notebook.add(self._build_forms_tab(notebook), text="表單")
        notebook.add(self._build_run_tab(notebook), text="自動流程")

        output_frame = ttk.LabelFrame(container, text="輸出", padding=(10, 6))
        output_frame.pack(fill=tk.BOTH, expand=True, pady=(12, 0))

        self._output = tk.Text(output_frame, height=12, wrap="word", state=tk.DISABLED)
        self._output.pack(side=tk.LEFT, fill=tk.BOTH, expand=True)

        scrollbar = ttk.Scrollbar(output_frame, orient="vertical", command=self._output.yview)
        scrollbar.pack(side=tk.RIGHT, fill=tk.Y)
        self._output.configure(yscrollcommand=scrollbar.set)

        controls = ttk.Frame(container)
        controls.pack(fill=tk.X, pady=(8, 0))
        ttk.Button(controls, text="清除輸出", command=self._clear_output).pack(side=tk.RIGHT)

    def _build_login_tab(self, parent: "tk.Misc"):
        frame = ttk.Frame(parent, padding=12)
        frame.columnconfigure(1, weight=1)

        self._add_labeled_entry(frame, "基礎網址", self._base_url, row=0)
        self._add_labeled_entry(frame, "逾時秒數", self._timeout, row=1)
        self._add_labeled_entry(frame, "帳號", self._login_account, row=2)
        self._add_labeled_entry(frame, "密碼", self._login_password, row=3, show="*")
        self._add_labeled_entry(frame, "登入頁面", self._login_page, row=4)
        self._add_labeled_entry(frame, "額外欄位 (key=value)", self._login_extra, row=5)

        dump_checkbox = ttk.Checkbutton(frame, text="顯示回應 HTML", variable=self._login_dump_html)
        dump_checkbox.grid(row=6, column=1, sticky="w", pady=(4, 8))

        ttk.Button(frame, text="執行登入", command=self._on_login).grid(
            row=7, column=1, sticky="e", pady=(8, 0)
        )

        return frame

    def _build_forms_tab(self, parent: "tk.Misc"):
        frame = ttk.Frame(parent, padding=12)
        frame.columnconfigure(1, weight=1)

        self._add_labeled_entry(frame, "基礎網址", self._base_url, row=0)
        self._add_labeled_entry(frame, "逾時秒數", self._timeout, row=1)
        self._add_labeled_entry(frame, "檢視網址", self._forms_url, row=2)

        include_checkbox = ttk.Checkbutton(
            frame,
            text="僅顯示包含密碼欄位的表單",
            variable=self._forms_include_password,
        )
        include_checkbox.grid(row=3, column=1, sticky="w", pady=(4, 8))

        ttk.Button(frame, text="擷取表單", command=self._on_dump_forms).grid(
            row=4, column=1, sticky="e", pady=(8, 0)
        )

        return frame

    def _build_run_tab(self, parent: "tk.Misc"):
        frame = ttk.Frame(parent, padding=12)
        frame.columnconfigure(1, weight=1)

        self._add_labeled_entry(frame, "基礎網址", self._base_url, row=0)
        self._add_labeled_entry(frame, "逾時秒數", self._timeout, row=1)

        ttk.Label(frame, text="設定檔").grid(row=2, column=0, sticky="w", pady=(6, 0))
        entry = ttk.Entry(frame, textvariable=self._run_config_path)
        entry.grid(row=2, column=1, sticky="ew", pady=(6, 0))
        ttk.Button(frame, text="瀏覽...", command=self._browse_config).grid(row=2, column=2, padx=(6, 0))

        verbose_checkbox = ttk.Checkbutton(frame, text="輸出完整 HTML", variable=self._run_verbose)
        verbose_checkbox.grid(row=3, column=1, sticky="w", pady=(4, 8))

        ttk.Button(frame, text="執行流程", command=self._on_run_config).grid(
            row=4, column=1, sticky="e", pady=(8, 0)
        )

        return frame

    def _add_labeled_entry(
        self,
        parent: "tk.Misc",
        label: str,
        variable: "tk.Variable",
        *,
        row: int,
        show: str | None = None,
    ) -> None:
        ttk.Label(parent, text=label).grid(row=row, column=0, sticky="w", pady=4)
        entry = ttk.Entry(parent, textvariable=variable, show=show)
        entry.grid(row=row, column=1, sticky="ew", pady=4)

    # ------------------------------------------------------------------
    # Event handlers
    # ------------------------------------------------------------------
    def _on_login(self) -> None:
        account = self._login_account.get().strip()
        password = self._login_password.get()
        if not account or not password:
            self._notify_error("請輸入帳號與密碼後再嘗試登入。")
            return

        args = argparse.Namespace(
            base_url=self._normalize_text(self._base_url.get(), cli.DEFAULT_BASE_URL),
            timeout=self._parse_timeout(self._timeout.get()),
            account=account,
            password=password,
            login_page=self._normalize_optional(self._login_page.get()),
            extra=self._split_extra(self._login_extra.get()),
            dump_html=bool(self._login_dump_html.get()),
        )
        self._run_command("登入", cli.command_login, args)

    def _on_dump_forms(self) -> None:
        args = argparse.Namespace(
            base_url=self._normalize_text(self._base_url.get(), cli.DEFAULT_BASE_URL),
            timeout=self._parse_timeout(self._timeout.get()),
            url=self._normalize_optional(self._forms_url.get()),
            include_password=bool(self._forms_include_password.get()),
        )
        self._run_command("擷取表單", cli.command_dump_forms, args)

    def _on_run_config(self) -> None:
        config_path = self._normalize_optional(self._run_config_path.get())
        if not config_path:
            self._notify_error("請先選擇設定檔再執行流程。")
            return

        args = argparse.Namespace(
            base_url=self._normalize_text(self._base_url.get(), cli.DEFAULT_BASE_URL),
            timeout=self._parse_timeout(self._timeout.get()),
            config=config_path,
            verbose=bool(self._run_verbose.get()),
        )
        self._run_command("執行流程", cli.command_run_config, args)

    def _browse_config(self) -> None:
        if filedialog is None:  # pragma: no cover - depends on Tk availability
            self._notify_error("目前環境不支援檔案對話框。")
            return

        path = filedialog.askopenfilename(
            title="選擇設定檔",
            filetypes=(("JSON", "*.json"), ("所有檔案", "*.*")),
        )
        if path:
            self._run_config_path.set(path)

    # ------------------------------------------------------------------
    # Command execution helpers
    # ------------------------------------------------------------------
    def _run_command(
        self,
        label: str,
        func: Callable[[argparse.Namespace], int],
        args: argparse.Namespace,
    ) -> None:
        self._append_output(f"[{label}] 開始執行...\n")

        def worker() -> None:
            buffer = io.StringIO()
            with contextlib.redirect_stdout(buffer), contextlib.redirect_stderr(buffer):
                try:
                    exit_code = func(args)
                except Exception as exc:  # pragma: no cover - surfaced via UI
                    exit_code = 1
                    buffer.write(f"Error: {exc}\n")
            output = buffer.getvalue().strip()
            self._root.after(0, lambda: self._finalize_command(label, exit_code, output))

        threading.Thread(target=worker, daemon=True).start()

    def _finalize_command(self, label: str, exit_code: int, output: str) -> None:
        if output:
            self._append_output(f"{output}\n")
        status = "成功" if exit_code == 0 else f"失敗 (代碼 {exit_code})"
        self._append_output(f"[{label}] {status}\n")
        if exit_code != 0:
            self._notify_error(f"{label} 執行失敗，詳情請見輸出區。")

    def _append_output(self, text: str) -> None:
        self._output.configure(state=tk.NORMAL)
        self._output.insert(tk.END, text)
        self._output.see(tk.END)
        self._output.configure(state=tk.DISABLED)

    def _clear_output(self) -> None:
        self._output.configure(state=tk.NORMAL)
        self._output.delete("1.0", tk.END)
        self._output.configure(state=tk.DISABLED)

    def _notify_error(self, message: str) -> None:
        self._append_output(f"[錯誤] {message}\n")
        if messagebox is not None:
            messagebox.showerror("Ticket Bot", message, parent=self._root)

    # ------------------------------------------------------------------
    # Utility helpers
    # ------------------------------------------------------------------
    @staticmethod
    def _split_extra(value: str | None) -> List[str]:
        if not value:
            return []
        tokens: List[str] = []
        for part in value.replace("\n", " ").split(" "):
            part = part.strip()
            if part:
                tokens.append(part)
        return tokens

    @staticmethod
    def _normalize_text(value: str | None, default: str) -> str:
        text = (value or "").strip()
        return text or default

    @staticmethod
    def _normalize_optional(value: str | None) -> str | None:
        text = (value or "").strip()
        return text or None

    @staticmethod
    def _parse_timeout(value: str | None) -> float:
        text = (value or "").strip()
        if not text:
            return cli.DEFAULT_TIMEOUT
        try:
            return float(text)
        except ValueError:
            return cli.DEFAULT_TIMEOUT

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------
    def run(self) -> None:
        if self._owns_root:
            self._root.mainloop()


def run_gui() -> None:
    """Launch the interactive GUI application."""

    app = TicketBotGUI()
    app.run()


if __name__ == "__main__":  # pragma: no cover - manual execution
    run_gui()
