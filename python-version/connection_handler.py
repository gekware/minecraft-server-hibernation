import functools
import json
import socket
from logging import getLogger
from time import sleep
from typing import Optional

from .data_usage import DataUsageMonitor
from .minecraft_server_controller import MinecraftServerController
from .proxy import Proxy
from .thread_helpers import set_interval

logger = getLogger(__name__)


class ConnectionHandler:
    def __init__(self, controller: MinecraftServerController, data_monitor: DataUsageMonitor, listen_host: str,
                 listen_port: int, server_host, server_port, data_logging_interval: Optional[int]):
        self.data_monitor = data_monitor
        self.controller = controller
        self.listen_host = listen_host
        self.listen_port = listen_port
        self.server_host = server_host
        self.server_port = server_port

        if data_logging_interval:
            def log_data_usage():
                logger.debug('{:.3f}KB/s'.format(self.data_monitor.kilobytes_per_second))

            set_interval(log_data_usage, data_logging_interval, thread_name="DataUsageLogging")

    def setup_player_counting_proxy(self, client: socket.socket, server: socket.socket):
        proxy = Proxy(server, client, self.data_monitor)

        @proxy.before_client_to_server
        def player_joins():
            self.controller.player_joined()

        @proxy.after_client_to_server
        def player_leaves():
            self.controller.player_left()

        proxy.start()

    def handle_connection(self, *, debug: bool):
        server_host = self.server_host
        server_port = self.server_port
        listen_port = self.listen_port
        logger.info('*** listening for new clients to connect...')
        client_socket, client_address = self.listen_socket.accept()  # blocking
        if debug:
            logger.debug(f'*** from {client_address[0]}:{listen_port} to {server_host}:{server_port}')
        if self.controller.server_is_online:
            server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            server_socket.connect((server_host, server_port))
            self.setup_player_counting_proxy(client_socket, server_socket)
        else:
            connection_data_recv = client_socket.recv(64)
            if connection_data_recv[-1] == 2:  # Final byte \x02 indicates a player join request
                self.handle_join_attempt(client_address, client_socket)
            elif connection_data_recv[-1] == 1:  # Final byte \x01 indicates a server info request
                self.handle_server_info_request(client_address)
            client_socket.shutdown(1)  # sends FIN to client
            client_socket.close()

    def handle_server_info_request(self, client_address):
        if self.controller.server_is_offline:
            logger.info(f'Unknown player requested server info from {client_address[0]}')
        if self.controller.server_is_starting:
            logger.info(f'Unknown player requested server info from {client_address[0]} during server startup')

    def handle_join_attempt(self, client_address, client_socket):
        player_data_recv = client_socket.recv(64)  # here it's reading an other packet containing the player name
        player_name = player_data_recv[3:].decode('utf-8', errors='replace')
        if self.controller.server_is_offline:
            logger.info(f"{player_name} tried to join from {client_address[0]}, starting server.")
            self.controller.start_minecraft_server()
        if self.controller.server_is_starting:
            logger.info(f"{player_name} tried to join from {client_address[0]} during server startup.")
            sleep(0.01)  # necessary otherwise it could throw an error:
            # Internal Exception: io.netty.handler.codec.Decoder.Exception java.lang.NullPointerException
            # the padding to 88 chars is important, otherwise some clients will fail to interpret
            # (byte 0x0a (equal to \n or new line) is used to put the phrase in the center of the screen)
            time_left = self.controller._time_left_until_up.value  # TODO Can this be abstracted ?
            display_text = f"Server is starting. Please wait. Time left: {time_left} seconds".ljust(88, '\x0a')
            display_json = json.dumps(dict(text=display_text))
            packet_prefix = "e\0c"  # TODO Find source for this prefix
            packet_contents = f"{packet_prefix}{display_json}".encode()
            client_socket.sendall(packet_contents)

    @property
    @functools.lru_cache
    def listen_socket(self):
        listen_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        listen_socket.setblocking(True)
        listen_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)  # Prevents errno 98 address already in use
        listen_socket.bind((self.listen_host, self.listen_port))
        listen_socket.listen(5)
        return listen_socket
