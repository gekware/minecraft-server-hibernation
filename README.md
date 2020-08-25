# Minecraft Vanilla Server Hibernation (Python - Go)

![logo](https://user-images.githubusercontent.com/53654579/90397372-09a9df80-e098-11ea-925c-29e9bdfc0b48.png)

version 6.9 (Python - Go)

concept, early-code and lastest improvements by [gekigek99](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)<br/>
contributor (advanced-code) by [najtin](https://github.com/najtin/minecraft-server-hibernation)<br/>
derived from [supernifty](https://github.com/supernifty/port-forwarder)<br/>

### INSTRUCTIONS:
This is a simple Python script to start a minecraft server on request and stop it when there are no player online.
How to use:
1. Install and run your desiered minecraft server
2. The server jar file should be as specified in the script
3. Check the server-port parameter in "server.properties" (it should be 25565)
4. Edit the paramters in the script as needed:
    - startMinecraftServerLin or startMinecraftServerWin
    - stopMinecraftServerLin or stopMinecraftServerWin (should already be good for most uses)
    - minecraftServerStartupTime
    - timeBeforeStoppingEmptyServer
5. run the script at reboot
6. you can connect to the server through port 25555

### DEFINITIONS:
Commands to start and stop minecraft server:
```Python
# only text in parethesis needs to be modified
startMinecraftServerLin = "cd {PATH/TO/SERVERFOLDER}; screen -dmS minecraftServer java {-Xmx1024M} {-Xms1024M} -jar {server.jar} nogui"
stopMinecraftServerLin = "screen -S minecraftServer -X stuff 'stop\\n'"
startMinecraftServerWin = "java {-Xmx1024M} {-Xms1024M} -jar {server.jar} nogui"
stopMinecraftServerWin = "stop"
```
Personally I set up a systemctl minecraft server service (called "minecraft-server") therefore I use:
```Python
startMinecraftServerLin = "sudo systemctl start minecraft-server"
stopMinecraftServerLin = "sudo systemctl stop minecraft-server"
```
If you are the first to access to minecraft world you will have to wait *30 seconds* and then try to connect again.
```Python
minecraftServerStartupTime = 30         #any parameter more than 10s is recommended
```
*120 seconds* is the time (after the last player disconnected) that the script waits before shutting down the minecraft server
```Python
timeBeforeStoppingEmptyServer = 120     #any parameter more than 60s is recommended
```


#### If you like what I do please consider having a cup of coffee with me at: https://www.buymeacoffee.com/gekigek99
