#!/usr/bin/env python3
"""
minecraft-vanilla_server_hibernation.py is used to start and stop automatically a vanilla minecraft server
Copyright (C) 2020  gekigek99
v4.2 (Python)
visit my github page: https://github.com/gekigek99
If you like what I do please consider having a cup of coffee with me at: https://www.buymeacoffee.com/gekigek99

Modified by dangercrow https://github.com/dangercrow
"""
import json
import os
import socket
from argparse import ArgumentParser
from threading import Timer
from time import sleep

from thread_helpers import set_interval
from .atomic_integer import AtomicInteger
from .data_usage import DataUsageMonitor
from .inhibitors import PlayerBasedWinInhibitor
from .proxy import Proxy
from .server_state import ServerState, ServerStateTracker

# ------------------------modify-------------------------------#

START_MINECRAFT_SERVER = 'cd PATH/TO/SERVERFOLDER; screen -dmS minecraftSERVER nice -19 java -jar minecraft_server.jar'  # set command to start minecraft-server service
STOP_MINECRAFT_SERVER = "screen -S minecraftSERVER -X stuff 'stop\\n'"  # set command to stop minecraft-server service

MINECRAFT_SERVER_STARTUPTIME = 20  # time the server needs until it is fully started
TIME_BEFORE_STOPPING_EMPTY_SERVER = 60  # time the server waits for clients to connect then it issues the stop command to server

# ---------------------do not modify---------------------------#

data_monitor = DataUsageMonitor()
server_status_tracker = ServerStateTracker()

players = AtomicInteger()
recent_activity = AtomicInteger()
timelefttillup = AtomicInteger(MINECRAFT_SERVER_STARTUPTIME)


def register_check_to_stop_empty_minecraft_server(time_until_check=TIME_BEFORE_STOPPING_EMPTY_SERVER):
    def stop_empty_minecraft_server():
        recent_activity.dec()
        if recent_activity.value > 0 or players.value > 0 or server_status_tracker.state == ServerState.OFFLINE:
            return
        server_status_tracker.state = ServerState.OFFLINE
        os.system(STOP_MINECRAFT_SERVER)
        print('MINECRAFT SERVER IS SHUTTING DOWN!')
        timelefttillup.value = MINECRAFT_SERVER_STARTUPTIME

    Timer(time_until_check, stop_empty_minecraft_server, ()).start()


def start_minecraft_server():
    if server_status_tracker.state != ServerState.OFFLINE:
        return
    server_status_tracker.state = ServerState.STARTING
    os.system(START_MINECRAFT_SERVER)
    print('MINECRAFT SERVER IS STARTING!')
    players.value = 0

    def _set_server_status_online():
        server_status_tracker.state = ServerState.ONLINE
        print('MINECRAFT SERVER IS UP!')
        recent_activity.inc()
        register_check_to_stop_empty_minecraft_server()

    def _update_timeleft():
        if timelefttillup.value > 0:
            timelefttillup.dec()

    set_interval(_update_timeleft, 1)

    Timer(MINECRAFT_SERVER_STARTUPTIME, _set_server_status_online).start()


def setup_player_counting_proxy(client, server):
    proxy = Proxy(server, client, data_monitor)

    @proxy.before_client_to_server
    def player_joins():
        players.inc()
        print(f"A PLAYER JOINED THE SERVER! - {players} players online")

    @proxy.after_client_to_server
    def player_leaves():
        players.dec()
        print(f"A PLAYER LEFT THE SERVER! - {players} players remaining")
        recent_activity.inc()
        register_check_to_stop_empty_minecraft_server()

    proxy.start()


def main(*, debug, listen_host, listen_port, server_host, server_port, data_usage_log_interval):
    print('minecraft-vanilla-server-hibernation v4.2 (Python)')
    print('Copyright (C) 2020 gekigek99')
    print('visit my github page for updates: https://github.com/gekigek99')
    set_interval(lambda: PlayerBasedWinInhibitor.with_players(players), 1, thread_name="WinInhibitor")
    dock_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    dock_socket.setblocking(True)
    dock_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)  # to prevent errno 98 address already in use
    dock_socket.bind((listen_host, listen_port))
    dock_socket.listen(5)
    print('*** listening for new clients to connect...')

    if debug:
        set_interval(lambda: print('{:.3f}KB/s'.format(data_monitor.kilobytes_per_second)), data_usage_log_interval,
                     thread_name="DataUsageLogging")
    while True:
        try:
            handle_connection(dock_socket, server_host, server_port, listen_port, debug)
        except Exception as e:
            print(f"Exception in main(): {e}")


def handle_connection(dock_socket: socket.socket, server_host: str, server_port: int, listen_port: int, debug: bool):
    client_socket, client_address = dock_socket.accept()  # blocking
    if debug:
        print(f'*** from {client_address[0]}:{listen_port} to {server_host}:{server_port}')
    if server_status_tracker.state == ServerState.OFFLINE or server_status_tracker.state == ServerState.STARTING:
        connection_data_recv = client_socket.recv(64)
        if connection_data_recv[-1] == 2:  # Final byte \x02 indicates a player join request
            handle_join_attempt(client_address, client_socket)
        elif connection_data_recv[-1] == 1:  # Final byte \x01 indicates a server info request
            handle_server_info_request(client_address)
        client_socket.shutdown(1)  # sends FIN to client
        client_socket.close()
    elif server_status_tracker.state == ServerState.ONLINE:
        server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        server_socket.connect((server_host, server_port))
        setup_player_counting_proxy(client_socket, server_socket)


def handle_server_info_request(client_address):
    if server_status_tracker.state == ServerState.OFFLINE:
        print(f'Unknown player requested server info from {client_address[0]}')
    if server_status_tracker.state == ServerState.STARTING:
        print(f'Unknown player requested server info from {client_address[0]} during server startup')


def handle_join_attempt(client_address, client_socket):
    player_data_recv = client_socket.recv(64)  # here it's reading an other packet containing the player name
    player_name = player_data_recv[3:].decode('utf-8', errors='replace')
    if server_status_tracker.state == ServerState.OFFLINE:
        print(f"{player_name} tried to join from {client_address[0]}, starting server.")
        start_minecraft_server()
    if server_status_tracker.state == ServerState.STARTING:
        print(f"{player_name} tried to join from {client_address[0]} during server startup.")
        sleep(0.01)  # necessary otherwise it could throw an error:
        # Internal Exception: io.netty.handler.codec.Decoder.Exception java.lang.NullPointerException
        # the padding to 88 chars is important, otherwise some clients will fail to interpret
        # (byte 0x0a (equal to \n or new line) is used to put the phrase in the center of the screen)
        display_text = f"Server is starting. Please wait. Time left: {timelefttillup} seconds".ljust(88, '\x0a')
        display_json = json.dumps(dict(text=display_text))
        packet_prefix = "e\0c"
        packet_contents = f"{packet_prefix}{display_json}".encode()
        client_socket.sendall(packet_contents)


if __name__ == '__main__':
    parser = ArgumentParser()
    parser.add_argument("--listen-host", default="0.0.0.0", help="The host on which the client should listen")
    parser.add_argument("--listen-port", type=int, default=25555, help="The port on which the client should listen")
    parser.add_argument("--server-host", default="0.0.0.0", help="The host on which the Minecraft server runs")
    parser.add_argument("--server-port", type=int, default=25565, help="The port on which the Minecraft server runs")
    parser.add_argument("--debug", action="set_true", default=False, help="If set, print additional debug information")
    parser.add_argument("--debug-data-usage-log-interval", type=int, default=3, help="Debug log frequency")

    args = parser.parse_args()

    main(
        debug=args.debug,
        listen_host=args.listen_host,
        listen_port=args.listen_port,
        server_host=args.server_host,
        server_port=args.server_port,
        data_usage_log_interval=args.debug_data_usage_log_interval,
    )
