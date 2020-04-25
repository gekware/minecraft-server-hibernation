import os
from threading import Timer

from atomic_integer import AtomicInteger
from inhibitors import PlayerBasedWinInhibitor
from server_state import ServerState, ServerStateTracker
from thread_helpers import set_interval

START_MINECRAFT_SERVER = 'cd PATH/TO/SERVERFOLDER; screen -dmS minecraftSERVER nice -19 java -jar minecraft_server.jar'  # set command to start minecraft-server service
STOP_MINECRAFT_SERVER = "screen -S minecraftSERVER -X stuff 'stop\\n'"  # set command to stop minecraft-server service


class MinecraftServerController:
    def __init__(self, *, expected_startup_time, idle_time_until_shutdown):
        self.expected_startup_time = expected_startup_time
        self.idle_time_until_shutdown = idle_time_until_shutdown
        self._time_left_until_up = AtomicInteger(self.expected_startup_time)
        self._recent_activity = AtomicInteger()
        self._players = AtomicInteger()
        self._server_status_tracker = ServerStateTracker()
        set_interval(lambda: PlayerBasedWinInhibitor.with_players(self._players), 1, thread_name="WinInhibitor")

    def start_minecraft_server(self):
        if self._server_status_tracker.state != ServerState.OFFLINE:
            return
        self._server_status_tracker.state = ServerState.STARTING
        os.system(START_MINECRAFT_SERVER)
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

        Timer(self.expected_startup_time, _set_server_status_online).start()

    def register_check_to_stop_empty_minecraft_server(self):
        def stop_empty_minecraft_server():
            self._recent_activity.dec()
            if self._recent_activity.value > 0 or self._players.value > 0 or self.server_is_offline:
                return
            self._server_status_tracker.state = ServerState.OFFLINE
            os.system(STOP_MINECRAFT_SERVER)
            print('MINECRAFT SERVER IS SHUTTING DOWN!')
            self._time_left_until_up.value = self.expected_startup_time

        Timer(self.idle_time_until_shutdown, stop_empty_minecraft_server, ()).start()

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
