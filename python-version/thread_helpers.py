from threading import Event, Thread
from typing import Callable


def set_interval(f: Callable, interval: float, *, thread_name=None):
    stop_event = Event()

    def thread_fn():
        while not stop_event.wait(interval):
            f()

    Thread(target=thread_fn, name=thread_name).start()
    return stop_event
