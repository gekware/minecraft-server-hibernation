from socket import socket
from threading import Thread
from typing import Callable, Optional

from data_usage import DataUsageMonitor


class Proxy:
    def __init__(self, server: socket, client: socket, data_monitor: DataUsageMonitor):
        self.server = server
        self.client = client
        self.data_monitor = data_monitor

        self.before_client_to_server_callback = None
        self.after_client_to_server_callback = None
        self.before_server_to_client_callback = None
        self.after_server_to_client_callback = None

        self._server_to_client_thread: Optional[Thread] = None
        self._client_to_server_thread: Optional[Thread] = None

    def before_client_to_server(self, f: Callable[[], None]):
        assert self.before_client_to_server_callback is None
        self.before_client_to_server_callback = f

    def after_client_to_server(self, f: Callable[[], None]):
        assert self.after_client_to_server_callback is None
        self.before_client_to_server_callback = f

    def before_server_to_client(self, f: Callable[[], None]):
        assert self.before_server_to_client_callback is None
        self.before_client_to_server_callback = f

    def after_server_to_client(self, f: Callable[[], None]):
        assert self.after_server_to_client_callback is None
        self.before_client_to_server_callback = f

    def start(self):
        self._server_to_client_thread = Thread(target=self.server_to_client, name="ServerToClient")
        self._client_to_server_thread = Thread(target=self.client_to_server, name="ClientToServer")

        self._server_to_client_thread.start()
        self._client_to_server_thread.start()

    def client_to_server(self):
        if self.before_client_to_server_callback:
            self.before_client_to_server_callback()
        Proxy.forward_sync(self.client, self.server, self.data_monitor)
        if self.after_client_to_server_callback:
            self.after_client_to_server_callback()

    def server_to_client(self):
        if self.before_server_to_client_callback:
            self.before_server_to_client_callback()
        Proxy.forward_sync(self.server, self.client, self.data_monitor)
        if self.after_server_to_client_callback:
            self.after_server_to_client_callback()

    @staticmethod
    def forward_sync(source: socket, destination: socket, data_monitor: DataUsageMonitor):
        source.settimeout(60)
        destination.settimeout(60)
        try:
            while True:
                data = source.recv(1024)
                if not data:  # if there is no data stop listening, this means the socket is closed
                    break
                destination.sendall(data)
                data_monitor.used_bytes(len(data))
        except IOError as e:
            if e.errno == 32:  # user/server disconnected normally. has to be caught, because there is a race condition
                return  # when trying to check if destination.recv does return data
            print(f"IOError in forward(): {e}")
        except Exception as e:
            print(f"Exception in forward(): {e}")
