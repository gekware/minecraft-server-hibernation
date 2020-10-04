# Minecraft Vanilla Server Hibernation (Python - Go)

[![mvsh - license](https://img.shields.io/github/license/gekigek99/minecraft-vanilla-server-hibernation?color=6fff00)](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)
[![mvsh - stars](https://img.shields.io/github/stars/gekigek99/minecraft-vanilla-server-hibernation?color=ffbd19)](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)
[![mvsh - release](https://img.shields.io/github/release/gekigek99/minecraft-vanilla-server-hibernation?color=05aefc)](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)  

[![mvsh - logo](https://user-images.githubusercontent.com/53654579/90397372-09a9df80-e098-11ea-925c-29e9bdfc0b48.png)](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)  

Version 9.9 (Python - Go)

-----

### INSTRUCTIONS:
This is a simple Python script to start a minecraft server on request and stop it when there are no player online.
How to use:
1. Install your desired minecraft server
2. "server-port" parameter in "server.properties" should be 25565
3. Edit the parameters in config.json as needed (*check definitions*):
    - startMinecraftServerLin or startMinecraftServerWin
    - stopMinecraftServerLin or stopMinecraftServerWin
    - minecraftServerStartupTime
    - timeBeforeStoppingEmptyServer 
4. on the server: open port 25555 (example: [ufw firewall](https://www.configserverfirewall.com/ufw-ubuntu-firewall/ubuntu-firewall-open-port/))
5. on the router: forward port 25555 to server ([tutorial](https://www.wikihow.com/Open-Ports#Opening-Router-Firewall-Ports))
6. you can connect to the server through port 25555

(remember to run the script at reboot)

### DEFINITIONS:
Commands to start and stop minecraft server:
```yaml
# only text in parethesis needs to be modified
"startMinecraftServerLin": "cd {PATH/TO/SERVERFOLDER}; screen -dmS minecraftServer java {-Xmx1024M} {-Xms1024M} -jar {server.jar} nogui",
"stopMinecraftServerLin": "screen -S minecraftServer -X stuff 'stop\\n'",
"startMinecraftServerWin": "java {-Xmx1024M} {-Xms1024M} -jar {server.jar} nogui",
"stopMinecraftServerWin": "stop",

# if you are on linux you can access the minecraft server console with "sudo screen -r minecraftServer"
```
Personally I set up a systemctl minecraft server service (called "minecraft-server") therefore I use:
```yaml
"startMinecraftServerLin": "sudo systemctl start minecraft-server",
"stopMinecraftServerLin": "sudo systemctl stop minecraft-server",
```
If you are the first to access to minecraft world you will have to wait *30 seconds* and then try to connect again.
```yaml
"minecraftServerStartupTime": 30,         #any parameter more than 10s is recommended
```
*120 seconds* is the time (after the last player disconnected) that the script waits before shutting down the minecraft server
```yaml
"timeBeforeStoppingEmptyServer": 120     #any parameter more than 60s is recommended
```  

-----
### CREDITS:  

Author: [gekigek99](https://github.com/gekigek99)  
Contributors: [najtin](https://github.com/najtin/minecraft-server-hibernation)  
Docker branch: [lubocode](https://github.com/gekigek99/minecraft-vanilla-server-hibernation/tree/docker)  

#### If you like what I do please consider having a cup of coffee with me at:  

<a href="https://www.buymeacoffee.com/gekigek99" target="_blank"><img src="https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png" alt="Buy Me A Coffee" style="height: 41px !important;width: 174px !important;box-shadow: 0px 3px 2px 0px rgba(190, 190, 190, 0.5) !important;-webkit-box-shadow: 0px 3px 2px 0px rgba(190, 190, 190, 0.5) !important;" ></a>

#### And remember to give a star to this repository [here](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)!
