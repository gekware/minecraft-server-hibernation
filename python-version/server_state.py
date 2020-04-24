import enum
from enum import auto


class ServerState(enum):
    OFFLINE = auto()
    STARTING = auto()
    ONLINE = auto()


class ServerStateTracker:
    def __init__(self):
        self._state = ServerState.OFFLINE

    @property
    def state(self):
        return self._state

    @state.setter
    def state(self, state):
        self.state = state
