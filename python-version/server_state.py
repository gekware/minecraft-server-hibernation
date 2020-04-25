from enum import auto, Enum


class ServerState(Enum):
    OFFLINE = auto()
    STARTING = auto()
    ONLINE = auto()


class ServerStateTracker:
    def __init__(self):
        self.state: ServerState = ServerState.OFFLINE
