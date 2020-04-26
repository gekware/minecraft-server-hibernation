#!/usr/bin/env python3
"""
minecraft-vanilla-server-hibernation is used to cause a vanilla minecraft server to "hibernate" when no players are
online for a while. The server will then spin up again when a player attempts to connect.

Original work Copyright (C) 2020  gekigek99
Modifications Copyright (C) 2020 dangercrow
v5.0 (Python)
"""
from argparse import ArgumentParser
from logging import getLogger, basicConfig, INFO, DEBUG
from pathlib import Path

from .connection_handler import ConnectionHandler
from .data_usage import DataUsageMonitor
from .minecraft_server_controller import MinecraftServerController

if __name__ == '__main__':
    parser = ArgumentParser()
    parser.add_argument("--minecraft-server-path", type=Path, help="The working directory in which to run the server")
    parser.add_argument("--minecraft-server-startup-command", help="The command to launch the server",
                        default="nice -19 java -jar minecraft_server.jar")
    parser.add_argument("--minecraft-server-stop-commands", help="Repeatable. Commands to run to stop the server",
                        nargs='+', default=['stop'])
    parser.add_argument("--prevent-windows-from-sleeping", action="store_true",
                        help="Prevent the machine from sleeping whilst the server is live. "
                             "Windows specific, may raise errors on other platforms.")

    parser.add_argument("--listen-host", default="0.0.0.0", help="The host on which the client should listen")
    parser.add_argument("--listen-port", type=int, default=25555, help="The port on which the client should listen")

    parser.add_argument("--server-host", default="0.0.0.0", help="The host on which the Minecraft server runs")
    parser.add_argument("--server-port", type=int, default=25565, help="The port on which the Minecraft server runs")

    parser.add_argument("--expected-startup-time", type=int, default=20, help="How long the server takes to start")
    parser.add_argument("--idle-time-until-shutdown", type=int, default=60,
                        help="How long the server should remain up, with no players, before shutting down")

    parser.add_argument("--debug", action="store_true", default=False, help="If set, print additional debug information")
    parser.add_argument("--debug-data-usage-log-interval", type=int, default=3, help="Debug log frequency")

    args = parser.parse_args()

    basicConfig(level=DEBUG if args.debug else INFO)

    data_monitor = DataUsageMonitor()

    server_controller = MinecraftServerController(
        minecraft_server_path=args.minecraft_server_path,
        expected_startup_time=args.expected_startup_time,
        idle_time_until_shutdown=args.idle_time_until_shutdown,
        startup_command=args.minecraft_server_startup_command,
        minecraft_commands_to_run_to_stop=args.minecraft_server_stop_commands,
        prevent_windows_from_sleeping=args.prevent_windows_from_sleeping
    )
    handler = ConnectionHandler(
        controller=server_controller,
        data_monitor=data_monitor,
        listen_host=args.listen_host,
        listen_port=args.listen_port,
        server_host=args.server_host,
        server_port=args.server_port,
        data_logging_interval=args.debug_data_usage_log_interval
    )
    logger = getLogger("Main")
    logger.info('minecraft-vanilla-server-hibernation v5.0 (Python)')
    logger.info('Original work Copyright (C) 2020 gekigek99')
    logger.info('Modifications Copyright (C) 2020 dangercrow')
    logger.info('Available on GitHub with full history at: '
                'https://github.com/dangercrow/minecraft-vanilla-server-hibernation')

    while True:
        try:
            handler.handle_connection()
        except Exception as e:
            logger.exception(e)
