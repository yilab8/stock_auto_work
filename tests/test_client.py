import unittest
from unittest import mock

from ticketbot.client import KhamTicketClient, LoginResult, Page


class ClientLoginTestCase(unittest.TestCase):
    def setUp(self):
        self.client = KhamTicketClient("https://example.com/application/utk01/UTK0101_03.aspx")

    def test_successful_login_submits_credentials(self):
        login_html = """
        <form action="/login" method="post">
            <input type="text" name="txtAccount">
            <input type="password" name="txtPwd">
            <input type="hidden" name="__VIEWSTATE" value="abc">
            <input type="submit" value="登入">
        </form>
        """
        success_html = "<html>登入成功</html>"

        responses = iter(
            [
                Page(url="https://example.com/application/utk01/UTK0101_03.aspx", status=200, body=login_html),
                Page(url="https://example.com/member", status=200, body=success_html),
            ]
        )
        captured_requests = []

        def fake_open(request):
            captured_requests.append(request)
            return next(responses)

        with mock.patch.object(self.client, "_open", side_effect=fake_open):
            result = self.client.login("user123", "pass456")

        self.assertTrue(result.success)
        self.assertEqual(len(captured_requests), 2)
        self.assertEqual(captured_requests[0].get_method(), "GET")
        self.assertEqual(captured_requests[1].get_method(), "POST")
        self.assertIn(b"txtAccount=user123", captured_requests[1].data)
        self.assertIn(b"txtPwd=pass456", captured_requests[1].data)

    def test_login_failure_when_form_missing(self):
        login_html = "<html><body>No form here</body></html>"

        responses = iter(
            [
                Page(url="https://example.com/application/utk01/UTK0101_03.aspx", status=200, body=login_html),
            ]
        )

        with mock.patch.object(self.client, "_open", side_effect=lambda request: next(responses)):
            result = self.client.login("user", "pass")
        self.assertFalse(result.success)
        self.assertIn("Login form not found", result.message)

    def test_login_failure_detected_from_response_body(self):
        login_html = """
        <form action="/login" method="post">
            <input type="text" name="txtAccount">
            <input type="password" name="txtPwd">
        </form>
        """
        failure_html = "<html>登入失敗，密碼錯誤</html>"
        responses = iter(
            [
                Page(url="https://example.com/application/utk01/UTK0101_03.aspx", status=200, body=login_html),
                Page(url="https://example.com/application/utk01/UTK0101_03.aspx", status=200, body=failure_html),
            ]
        )

        with mock.patch.object(self.client, "_open", side_effect=lambda request: next(responses)):
            result = self.client.login("user", "pass")
        self.assertFalse(result.success)
        self.assertIn("Login rejected", result.message)


if __name__ == "__main__":
    unittest.main()
