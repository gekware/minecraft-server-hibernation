# minecraft-server-hibernation
concept and early-code by [gekigek99](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)
contributor (advanced-code) (big thanks) [najtin](https://github.com/najtin/minecraft-server-hibernation)
derived from [supernifty](https://github.com/supernifty/port-forwarder)

This is a simple Python script to start a minecraft server on request and stop it when there are no player online.
How to use:
1. Install and run your desiered minecraft server
2. Rename the minecraft-server-jar to 'minecraft_server.jar'
3. Change the port in 'server.properties' to 25555
4. Edited the paramters in the script as needed. 
5. run the script at reboot
6. you can connect to the server through port 25565

**IMPORTANT**	
If you are the first to access to minecraft world you will *have to wait 120 seconds* and then try to connect again.
```Python
MINECRAFT_SERVER_STARTUPTIME = 120 
```
After 120 seconds you have 240 to connect to the server before it is shutdown. 
```Python
TIME_BEFORE_STOPPING_EMPTY_SERVER = 240
```
You can change these parameters to fit your needs.
