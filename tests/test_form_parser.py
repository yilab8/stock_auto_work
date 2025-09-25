import unittest

from ticketbot.form_parser import FormParser


class FormParserTestCase(unittest.TestCase):
    def test_parse_basic_form(self):
        html = """
        <form action="/submit" method="post">
            <input type="text" name="username" value="">
            <input type="password" name="password">
            <input type="hidden" name="__VIEWSTATE" value="state">
            <select name="area">
                <option value="a">A</option>
                <option value="b" selected>B</option>
            </select>
            <textarea name="note">hello</textarea>
            <input type="submit" value="送出">
        </form>
        """

        parser = FormParser("https://example.com/login")
        form = parser.find_first(html)
        self.assertIsNotNone(form)
        action, method, fields = form.merged_with({})
        self.assertEqual(action, "https://example.com/submit")
        self.assertEqual(method, "POST")
        self.assertEqual(fields["username"], "")
        self.assertEqual(fields["password"], "")
        self.assertEqual(fields["__VIEWSTATE"], "state")
        self.assertEqual(fields["area"], "b")
        self.assertEqual(fields["note"].strip(), "hello")

    def test_find_first_with_password(self):
        html = """
        <form action="/one">
            <input name="foo">
        </form>
        <form action="/two">
            <input type="password" name="pwd">
        </form>
        """
        parser = FormParser("https://example.com/")
        form = parser.find_first(html, include_password=True)
        self.assertIsNotNone(form)
        self.assertEqual(form.action, "https://example.com/two")


if __name__ == "__main__":
    unittest.main()
