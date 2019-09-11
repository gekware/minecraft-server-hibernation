# minecraft-vanilla-server-hibernation
version 2.0
created by gekigek99

==========================================================================

Simple Python scripts to listen and forward network traffic (adapted manage minecraft server vanilla)

==========================================================================

Project derived from:	supernifty
link github:			https://github.com/supernifty/port-forwarder

==========================================================================

Usage
1: create a service for minecraft server and set it to not launch automatically at start-up

2: create a service for minecraft-vanilla_server_hibernation and set it to launch at start up

3: set on the .py file:
			START_MINECRAFT_SERVER	(example: 'sudo systemctl start minecraft-server')
			STOP_MINECRAFT_SERVER	(example: 'sudo systemctl stop minecraft-server')
			LISTEN_HOST				(example: "0.0.0.0")
			LISTEN_PORT				(example: 25555)
			TARGET_HOST				(example: "127.0.0.1")
			TARGET_PORT				(example: 25565)

4: DONE!

Note:	if you are the first to access to minecraft world you will have to wait 12 seconds
			(you can modify this if you want) to let the server load the world, and then retry to connect
			(to retry you have the amount of seconds specified in TIMEOUT_SOCKET)

==========================================================================

If you are planning to use this script for your server or you do modifications tell me, I would like to hear 
that all the hours I spent modifying this are put to good use by others!
