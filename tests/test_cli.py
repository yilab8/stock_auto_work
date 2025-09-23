import io
import json
import os
import tempfile
import unittest
from unittest import mock

from ticketbot import cli
from ticketbot.client import LoginResult, Page


class CLITestCase(unittest.TestCase):
    def test_command_login_success(self):
        fake_client = mock.Mock()
        fake_client.login.return_value = LoginResult(True, "Login succeeded", Page("https://example.com", 200, "OK"))

        with mock.patch("ticketbot.cli.KhamTicketClient", return_value=fake_client):
            parser = cli.build_parser()
            args = parser.parse_args(
                [
                    "login",
                    "--account",
                    "user",
                    "--password",
                    "pass",
                ]
            )
            with mock.patch("sys.stdout", new=io.StringIO()) as stdout:
                exit_code = args.func(args)

        self.assertEqual(exit_code, 0)
        self.assertIn("Login succeeded", stdout.getvalue())
        fake_client.login.assert_called_once()

    def test_command_run_config_executes_steps(self):
        fake_client = mock.Mock()
        login_result = LoginResult(True, "Login succeeded", Page("https://example.com/login", 200, "login"))
        fake_client.login.return_value = login_result
        fake_client.fetch.return_value = Page("https://example.com/event", 200, "<form></form>")

        form = mock.Mock()
        form.action = "https://example.com/submit"
        form.method = "POST"
        form.fields = {"a": "1"}
        form.field_types = {"a": "hidden"}

        with mock.patch("ticketbot.cli.FormParser") as parser_cls:
            parser_instance = parser_cls.return_value
            parser_instance.parse.return_value = [form]
            fake_client.submit.return_value = Page("https://example.com/done", 200, "Done")

            with tempfile.NamedTemporaryFile("w", delete=False, suffix=".json") as handle:
                json.dump(
                    {
                        "base_url": "https://example.com",
                        "login": {"account": "user", "password": "pass"},
                        "steps": [
                            {
                                "type": "submit",
                                "use_last_page": False,
                                "url": "https://example.com/event",
                                "form": {"action_contains": "submit"},
                                "overrides": {"a": "2"},
                            }
                        ],
                    },
                    handle,
                )
                temp_path = handle.name

            try:
                with mock.patch("ticketbot.cli.KhamTicketClient", return_value=fake_client):
                    parser = cli.build_parser()
                    args = parser.parse_args(["run", "--config", temp_path])
                    with mock.patch("sys.stdout", new=io.StringIO()):
                        exit_code = args.func(args)
            finally:
                os.unlink(temp_path)

        self.assertEqual(exit_code, 0)
        fake_client.submit.assert_called_once()
        fake_client.fetch.assert_called()


if __name__ == "__main__":
    unittest.main()
