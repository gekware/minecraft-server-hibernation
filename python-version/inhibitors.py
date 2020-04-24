from .atomic_integer import AtomicInteger


class WindowsInhibitor:
    ES_CONTINUOUS = 0x80000000
    ES_SYSTEM_REQUIRED = 0x00000001

    @staticmethod
    def inhibit():
        """Prevents Windows from going to sleep"""
        import ctypes
        print("Preventing Windows from going to sleep")
        es_flags = WindowsInhibitor.ES_CONTINUOUS | WindowsInhibitor.ES_SYSTEM_REQUIRED
        ctypes.windll.kernel32.SetThreadExecutionState(es_flags)

    @staticmethod
    def uninhibit():
        """Allows Windows to go to sleep"""
        import ctypes
        print("Allowing Windows to go to sleep")
        ctypes.windll.kernel32.SetThreadExecutionState(WindowsInhibitor.ES_CONTINUOUS)


class PlayerBasedWinInhibitor:
    @staticmethod
    def with_players(player_count: AtomicInteger):
        if player_count > 0:
            WindowsInhibitor.inhibit()
        else:
            WindowsInhibitor.uninhibit()
