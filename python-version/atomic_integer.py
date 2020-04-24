from functools import total_ordering
from threading import Lock


@total_ordering
class AtomicInteger:
    def __init__(self, value=0):
        self._value = value
        self._lock = Lock()

    def inc(self):
        with self._lock:
            self._value += 1
            return self._value

    def dec(self):
        with self._lock:
            self._value -= 1
            return self._value

    @property
    def value(self):
        with self._lock:
            return self._value

    @value.setter
    def value(self, v):
        with self._lock:
            self._value = v

    def __eq__(self, other):
        return self._value == other.value

    def __lt__(self, other):
        return self.value < other.value
