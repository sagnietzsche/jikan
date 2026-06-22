from time import monotonic

from textual.app import App, ComposeResult
from textual.widgets import Footer, Header, Static, Button
from textual.containers import Horizontal, ScrollableContainer
from textual.timer import Timer


def format_time(elapsed: float) -> str:
    total_centiseconds = int(elapsed * 100)
    centiseconds = total_centiseconds % 100
    total_seconds = total_centiseconds // 100
    seconds = total_seconds % 60
    total_minutes = total_seconds // 60
    minutes = total_minutes % 60
    hours = total_minutes // 60
    return f"{hours:02}:{minutes:02}:{seconds:02}.{centiseconds:02}"


class TimeDisplay(Static):
    pass


class StopWatch(Horizontal):
    def __init__(self) -> None:
        super().__init__()
        self.elapsed = 0.0
        self._started_at: float | None = None
        self._timer: Timer | None = None

    def compose(self) -> ComposeResult:
        yield Button("Start", variant="success", classes="start")
        yield Button("Stop", variant="error", classes="stop")
        yield Button("Reset", classes="reset")
        yield TimeDisplay(format_time(0), classes="time")

    def on_mount(self) -> None:
        self._timer = self.set_interval(1 / 30, self.refresh_time, pause=True)

    def on_button_pressed(self, event: Button.Pressed) -> None:
        event.stop()
        button = event.button
        if button.has_class("start"):
            self.start()
        elif button.has_class("stop"):
            self.stop()
        elif button.has_class("reset"):
            self.reset()

    def start(self) -> None:
        if self._started_at is None:
            self._started_at = monotonic()
            self.add_class("running")
            if self._timer is not None:
                self._timer.resume()
            self.refresh_time()

    def stop(self) -> None:
        if self._started_at is not None:
            self.elapsed += monotonic() - self._started_at
            self._started_at = None
            self.remove_class("running")
            if self._timer is not None:
                self._timer.pause()
            self.refresh_time()

    def reset(self) -> None:
        self.elapsed = 0.0
        if self._started_at is not None:
            self._started_at = monotonic()
        self.refresh_time()

    def refresh_time(self) -> None:
        elapsed = self.elapsed
        if self._started_at is not None:
            elapsed += monotonic() - self._started_at
        self.query_one(TimeDisplay).update(format_time(elapsed))


class Jikan(App):
    CSS_PATH = "stopwatch.css"
    BINDINGS = [("d", "toggle_dark_mode", "Toggle Dark Mode")]

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield Footer()
        yield ScrollableContainer(
            StopWatch(),
            StopWatch(),
            StopWatch(),
            id="stopwatches",
        )

    def action_toggle_dark_mode(self):
        self.theme = (
            "textual-dark" if self.theme == "textual-light" else "textual-light"
        )


if __name__ == "__main__":
    Jikan().run()
