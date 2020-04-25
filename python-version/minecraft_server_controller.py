import os
from pathlib import Path
from threading import Timer

from atomic_integer import AtomicInteger
from server_state import ServerState, ServerStateTracker
from thread_helpers import set_interval


class MinecraftServerController:
    def __init__(self, minecraft_server_path: Path, *, startup_command, minecraft_commands_to_run_to_stop, expected_startup_time, idle_time_until_shutdown, prevent_windows_from_sleeping):
        self._expected_startup_time = expected_startup_time
        self._idle_time_until_shutdown = idle_time_until_shutdown
        self._time_left_until_up = AtomicInteger(self._expected_startup_time)
        self._recent_activity = AtomicInteger()
        self._players = AtomicInteger()
        self._server_status_tracker = ServerStateTracker()

        self.start_minecraft_server_command = f'cd {minecraft_server_path.absolute()}; screen -dmS minecraftSERVER {startup_command}'
        stop_commands = '\\n'.join(minecraft_commands_to_run_to_stop)
        self.stop_minecraft_server_command = f"screen -S minecraftSERVER -X stuff '{stop_commands}\\n'"

        if prevent_windows_from_sleeping:
            from inhibitors import PlayerBasedWinInhibitor
            set_interval(lambda: PlayerBasedWinInhibitor.with_players(self._players), 1, thread_name="WinInhibitor")

    def start_minecraft_server(self):
        if self._server_status_tracker.state != ServerState.OFFLINE:
            return
        self._server_status_tracker.state = ServerState.STARTING

        os.system(self.start_minecraft_server_command)
        print('MINECRAFT SERVER IS STARTING!')
        self._players.value = 0

        def _set_server_status_online():
            self._server_status_tracker.state = ServerState.ONLINE
            print('MINECRAFT SERVER IS UP!')
            self._recent_activity.inc()
            self.register_check_to_stop_empty_minecraft_server()

        def _update_timeleft():
            if self._time_left_until_up.value > 0:
                self._time_left_until_up.dec()

        set_interval(_update_timeleft, 1)

        Timer(self._expected_startup_time, _set_server_status_online).start()

    def register_check_to_stop_empty_minecraft_server(self):
        def stop_empty_minecraft_server():
            self._recent_activity.dec()
            if self._recent_activity.value > 0 or self._players.value > 0 or self.server_is_offline:
                return
            self._server_status_tracker.state = ServerState.OFFLINE
            os.system(self.stop_minecraft_server_command)
            print('MINECRAFT SERVER IS SHUTTING DOWN!')
            self._time_left_until_up.value = self._expected_startup_time

        Timer(self._idle_time_until_shutdown, stop_empty_minecraft_server, ()).start()

    def player_left(self):
        self._players.dec()
        print(f"A PLAYER LEFT THE SERVER! - {self._players.value} players remaining")
        self._recent_activity.inc()
        self.register_check_to_stop_empty_minecraft_server()

    def player_joined(self):
        self._players.inc()
        print(f"A PLAYER JOINED THE SERVER! - {self._players.value} players online")

    @property
    def server_is_online(self):
        return self._server_status_tracker.state == ServerState.ONLINE

    @property
    def server_is_starting(self):
        return self._server_status_tracker.state == ServerState.STARTING

    @property
    def server_is_offline(self):
        return self._server_status_tracker.state == ServerState.OFFLINE
