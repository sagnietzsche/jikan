from textual.app import App, ComposeResult
from textual.widgets import Footer, Header, Static, Button


class TimeDisplay(Static):
    pass


class Stopwatch(Static):
    # custom stopwatch widget
    # define the compose method to define how it will layout things
    #
    def compose(self):
        yield Button("Start")
        yield Button("Stop")
        yield Button("Reset")
        yield TimeDisplay("00:00:00.00")


class Jikan(App):
    # KEY BINDIGNS
    BINDINGS = [("d", "toggle_dark_mode", "Toggle Dark Mode")]

    def compose(self) -> ComposeResult:
        # what widgets is this app composed of ?
        # yield Header(show_clock=True)
        # yield Footer()
        # yield Button("Start")
        # yield Button("Stop")
        yield Header(show_clock=True)
        yield Footer()

    def action_toggle_dark_mode(self):
        # name of the method shoule be prefixed with action
        # this is a action method
        # associated with the action toggle_dark_mode
        # toggle the dark mode
        self.theme = (
            "textual-dark" if self.theme == "textual-light" else "textual-light"
        )


if __name__ == "__main__":
    Jikan().run()
