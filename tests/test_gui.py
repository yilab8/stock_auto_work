import unittest
from unittest import mock

from ticketbot import cli
from ticketbot import gui


class TicketBotGUITestCase(unittest.TestCase):
    def test_run_gui_invokes_application(self):
        with mock.patch.object(gui, "TicketBotGUI") as app_cls:
            app_instance = app_cls.return_value
            gui.run_gui()
        app_cls.assert_called_once()
        app_instance.run.assert_called_once()

    def test_helper_parsers(self):
        self.assertEqual(gui.TicketBotGUI._split_extra(" a=1  b=2\nc=3 "), ["a=1", "b=2", "c=3"])
        self.assertEqual(gui.TicketBotGUI._split_extra(""), [])
        self.assertAlmostEqual(gui.TicketBotGUI._parse_timeout("1.5"), 1.5)
        self.assertEqual(gui.TicketBotGUI._parse_timeout(""), cli.DEFAULT_TIMEOUT)
        self.assertEqual(gui.TicketBotGUI._parse_timeout("abc"), cli.DEFAULT_TIMEOUT)


if __name__ == "__main__":
    unittest.main()
