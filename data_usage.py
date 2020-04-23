from collections import deque
from datetime import datetime


class DataUsageMonitor:
    def __init__(self, window_length: int = 3):
        self._data_packs = deque()
        self._window_length = window_length

        self.bytes_received_in_window = 0

    def used_bytes(self, byte_count: int):
        time_received = datetime.now()

        # Add new data to window
        self._data_packs.append((time_received, byte_count))
        self.bytes_received_in_window += byte_count

        while True:
            time_stored, byte_count = self._data_packs[0]  # Peek
            if time_received - time_stored > self._window_length:  # Data is older than window cares about
                self._data_packs.popleft()
                self.bytes_received_in_window -= byte_count
            else:  # Data is in window - all data to the right is also in window
                break

    @property
    def kilobytes_per_second(self) -> float:
        return self.bytes_received_in_window / self._window_length / 1024
