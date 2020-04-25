#!/usr/bin/env python3
"""
minecraft-vanilla_server_hibernation.py is used to start and stop automatically a vanilla minecraft server
Copyright (C) 2020  gekigek99
v4.2 (Python)
visit my github page: https://github.com/gekigek99
If you like what I do please consider having a cup of coffee with me at: https://www.buymeacoffee.com/gekigek99

Modified by dangercrow https://github.com/dangercrow
"""
from argparse import ArgumentParser

from .connection_handler import ConnectionHandler
from .data_usage import DataUsageMonitor
from .minecraft_server_controller import MinecraftServerController
from .thread_helpers import set_interval


def main(*, debug, listen_host, listen_port, server_host, server_port, data_usage_log_interval):
    print('minecraft-vanilla-server-hibernation v4.2 (Python)')
    print('Copyright (C) 2020 gekigek99')
    print('visit my github page for updates: https://github.com/gekigek99')

    data_monitor = DataUsageMonitor()
    server_controller = MinecraftServerController()
    connection_handler = ConnectionHandler(server_controller, data_monitor, listen_host, listen_port, server_host, server_port)
    print('*** listening for new clients to connect...')

    if debug:
        set_interval(lambda: print('{:.3f}KB/s'.format(data_monitor.kilobytes_per_second)), data_usage_log_interval,
                     thread_name="DataUsageLogging")
    while True:
        try:
            connection_handler.handle_connection(debug=debug)
        except Exception as e:
            print(f"Exception in main(): {e}")


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
