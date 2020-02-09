# minecraft-server-hibernation
done by [gekigek99](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)
derived from [supernifty](https://github.com/supernifty/port-forwarder)
contributor [najtin](https://github.com/najtin/minecraft-server-hibernation)

This is a simple Python script to start a minecraft server on request and stop it when there are no player online.
How to use:
1. Install and run your desiered minecraft server.
2. Rename the minecraft-server-jar to 'minecraft_server.jar'
3. Change the port in 'server.properties' to 25555
4. Edited the paramters in the script as needed. 
5. run the script

**IMPORTANT**	
If you are the first to access to minecraft world you will *have to wait 120 seconds*.
```Python
MINECRAFT_SERVER_STARTUPTIME = 120 
```
After that you have 240 to connect to the server before it is shutdown. 
```Python
TIMEOUT_SOCKET = 240
```
You can change these parameters to fit your needs.
