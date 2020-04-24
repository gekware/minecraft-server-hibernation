import enum
from enum import auto


class ServerState(enum):
    OFFLINE = auto()
    STARTING = auto()
    ONLINE = auto()
