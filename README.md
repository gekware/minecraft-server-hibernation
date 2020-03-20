# minecraft-vanilla-server-hibernation
version 2.1
created by gekigek99

==========================================================================

Simple Python scripts to listen and forward network traffic (adapted manage start and stop minecraft server vanilla to avoid wasting of resources on your server)

==========================================================================

Project derived from: [supernifty/port-forwarder](https://github.com/supernifty/port-forwarder)

==========================================================================

How to use:
1) create a service for minecraft server and set it to not launch automatically at start-up
2) create a service for minecraft-vanilla_server_hibernation and set it to launch at start up
3) set on the .py file:
	3.1) START_MINECRAFT_SERVER	(example: 'sudo systemctl start minecraft-server')
	3.2) STOP_MINECRAFT_SERVER	(example: 'sudo systemctl stop minecraft-server')
	3.3) LISTEN_HOST		(example: "0.0.0.0")
	3.4) LISTEN_PORT		(example: 25555)
	3.5) TARGET_HOST		(example: "127.0.0.1")
	3.6) TARGET_PORT		(example: 25565)
4) DONE!

Note:	if you are the first to access to minecraft world you will have to wait 12 seconds
			(you can modify this if you want) to let the server load the world, and then retry to connect
			(to retry you have the amount of seconds specified in TIMEOUT_SOCKET)

==========================================================================

If you are planning to use this script for your server or you do modifications tell me, I would like to hear 
that all the hours I spent modifying this are put to good use by others!
