# Minecraft Server Hibernation  

[![msh - release](https://img.shields.io/github/release/gekigek99/minecraft-server-hibernation?color=05aefc)](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)
[![msh - goreport](https://goreportcard.com/badge/github.com/gekigek99/minecraft-server-hibernation)](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)
[![msh - license](https://img.shields.io/github/license/gekigek99/minecraft-server-hibernation?color=6fff00)](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)
[![msh - stars](https://img.shields.io/github/stars/gekigek99/minecraft-server-hibernation?color=ffbd19)](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)

Avoid wasting of resources by **automatically start** your minecraft server when a player join **and stop** it when no one is online  
_(for vanilla/modded on linux/windows/macos)_  

<p align="center" >
    <a href="https://github.com/gekigek99/minecraft-server-hibernation" >
        <img src="https://user-images.githubusercontent.com/53654579/90397372-09a9df80-e098-11ea-925c-29e9bdfc0b48.png" >
    </a>
</p>

version: v2.3.3  
Copyright (C) 2019-2021 [gekigek99](https://github.com/gekigek99)  

Check the [releases](https://github.com/gekigek99/minecraft-server-hibernation/releases) to download the binaries (for linux, windows and macos)

_You can compile msh from the dev branch to access a more updated version but beware that it might still need testing_

-----
### PROGRAM COMPILATION:
This version was successfully compiled in go version 1.15  
Compilation procedure:
```
git clone https://github.com/gekigek99/minecraft-server-hibernation.git  
cd minecraft-server-hibernation/  
go build .
```
now you can run the msh executable

-----
### INSTRUCTIONS:
This is a Golang script to start a minecraft server on request and stop it when there are no players online.
How to use:
1. Install your desired minecraft server
2. Edit the parameters in config.json as needed (*check definitions*):
    - Folder
    - FileName
    - StartServer
    - StopServer
    - \* StopServerForce
    - \* HibernationInfo and StartingInfo
    - \* TimeBeforeStoppingEmptyServer
    - \* CheckForUpdates
3. \* put the frozen icon you want in "path/to/server.jar/folder" (must be 64x64 and called "server-icon-frozen.png")
4. on the router: forward port 25555 to server ([tutorial](https://www.wikihow.com/Open-Ports#Opening-Router-Firewall-Ports))
5. on the server: open port 25555 (example: [ufw firewall](https://www.configserverfirewall.com/ufw-ubuntu-firewall/ubuntu-firewall-open-port/))
6. run the msh executable
7. you can connect to the server through port 25555

\* = it's not compulsory to modify this parameter

(remember to run the set msh to run at reboot)

-----
### DEFINITIONS:
_only text in braces needs to be modified (remember to remove all braces)_

Location of server folder and executable:
```yaml
"Server":{
  "Folder": "{path/to/server/folder}",
  "FileName": "{server.jar}"
}
```
Commands to start and stop minecraft server:
```yaml
"Commands":{
  "StartServer": "java {-Xmx1024M} {-Xms1024M} -jar serverFileName nogui",
  "StopServer": "{stop}",
  "StopServerForce": ""
}

# the StopMinecraftServerForce parameter is used to execute special commands when a forced shutdown happens
```
Hibernation and warming up server description
```yaml
"HibernationInfo": "                   \u0026fserver status:\n                   \u0026b\u0026lHIBERNATING",
"StartingInfo": "                   \u0026fserver status:\n                    \u00266\u0026lWARMING UP"
```
*60 seconds* is the time (after the last player disconnected) that the script waits before shutting down the minecraft server
```yaml
"TimeBeforeStoppingEmptyServer": 60     #any parameter more than 30s is recommended
```
Set to false if you don't want to check for updates
```yaml
"CheckForUpdates": true
```
-----
### CREDITS:  

Author: [gekigek99](https://github.com/gekigek99)  
Contributors: [najtin](https://github.com/najtin/minecraft-server-hibernation) [f8ith](https://github.com/f8ith/minecraft-server-hibernation) [Br31zh](https://github.com/Br31zh/minecraft-server-hibernation) [someotherotherguy](https://github.com/someotherotherguy/minecraft-server-hibernation)  
Docker branch: [lubocode](https://github.com/gekigek99/minecraft-server-hibernation/tree/docker)  

_If you wish to contribute, please create a pull request using the dev branch as the base for your changes_

-----

<p align="center" >
    <a href="https://www.buymeacoffee.com/gekigek99" >
        <img src="https://user-images.githubusercontent.com/53654579/98535501-81963900-2286-11eb-94a4-359adb64afe2.png" >
    </a>
</p>

<h4 align="center" >
    Give a star to this repository on <a href="https://github.com/gekigek99/minecraft-server-hibernation" > github</a>!
</h4>

<p align="center" >
    <a href="https://github.com/gekigek99/minecraft-server-hibernation/stargazers" >
        <img src="https://reporoster.com/stars/gekigek99/minecraft-server-hibernation" >
    </a>
</p>
