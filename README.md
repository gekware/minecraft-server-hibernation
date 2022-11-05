# Minecraft Server Hibernation  

[![msh - loc](https://tokei.rs/b1/github/gekware/minecraft-server-hibernation)](https://github.com/gekware/minecraft-server-hibernation)
[![msh - release](https://img.shields.io/github/release/gekware/minecraft-server-hibernation?color=05aefc)](https://github.com/gekware/minecraft-server-hibernation/releases)
[![msh - goreport](https://goreportcard.com/badge/github.com/gekware/minecraft-server-hibernation)](https://goreportcard.com/report/github.com/gekware/minecraft-server-hibernation)
[![msh - license](https://img.shields.io/github/license/gekware/minecraft-server-hibernation?color=6fff00)](https://github.com/gekware/minecraft-server-hibernation/blob/master/LICENSE)
[![msh - stars](https://img.shields.io/github/stars/gekware/minecraft-server-hibernation?color=ffbd19)](https://github.com/gekware/minecraft-server-hibernation/stargazers)

Avoid wasting of resources by **automatically starting** your minecraft server when a player join **and stopping** it when no one is online  
_(for vanilla/modded on linux/windows/macos)_  

<p align="center" >
    <a href="https://github.com/gekware/minecraft-server-hibernation" >
        <img src="https://raw.githubusercontent.com/gekware/minecraft-server-hibernation/c6a80ea835bea9f9a795c0805ab0e99ba326273c/res/icon/msh.png" >
    </a>
</p>

version: v2.4.8  
Copyright (C) 2019-2022 [gekigek99](https://github.com/gekigek99)  

Check the [releases](https://github.com/gekware/minecraft-server-hibernation/releases) to download the binaries (for linux, windows and macos)

_You can compile msh from the dev branch to access a more updated version but beware that it might still need testing_

-----
### PROGRAM COMPILATION:
This version was successfully compiled in go version 1.15  
Compilation procedure:
```
git clone https://github.com/gekware/minecraft-server-hibernation.git  
cd minecraft-server-hibernation/  
go build .
```

-----
### INSTRUCTIONS:
1. Install your desired minecraft server
2. Edit the parameters in config file as needed (*check definitions*):
    - Folder
    - FileName
    - StartServerParam
    - StopServer
	- Whitelist
    - \* StopServerAllowKill
    - \* HibernationInfo and StartingInfo
    - \* TimeBeforeStoppingEmptyServer
    - \* NotifyUpdate
3. \* put the frozen icon you want in `path/to/server.jar/folder` (must be called `server-icon-frozen`, supported formats: `.png`, `.jpg`)
4. on the router (to which the server is connected): forward port 25555 to server ([tutorial](https://www.wikihow.com/Open-Ports#Opening-Router-Firewall-Ports))
5. on the server: open port 25555 (example: [ufw firewall](https://www.configserverfirewall.com/ufw-ubuntu-firewall/ubuntu-firewall-open-port/))
6. run the msh executable
7. you can connect to the server through port 25555

\* = it's not compulsory to modify this parameter

_remember to automatically run msh at reboot_

-----
### DEFINITIONS:
- _only text in braces needs to be modified (remember to remove all braces)_  
- _Some of these parameters can be configured with command-line arguments (--help to know which)_  

Location of server folder and executable. You can find protocol/version [here](https://wiki.vg/Protocol_version_numbers) (but msh should set them automatically):
```yaml
"Server": {
  "Folder": "{path/to/server/folder}"
  "FileName": "{server.jar}"
  "Protocol": 756
  "Version": "1.17.1"
}
```
Commands to start and stop minecraft server  
_StopServerAllowKill allows to kill the server after a certain amount of time (in seconds) when it's not responding_
```yaml
"Commands": {
  "StartServer": "java <Commands.StartServerParam> -jar <Server.FileName> nogui"
  "StartServerParam": "-Xmx1024M -Xms1024M"
  "StopServer": "stop"
  "StopServerAllowKill": 10	# set to 0 to disable
}
```
Set the logging level for debug purposes
```yaml
"Debug": 1
# 0 - NONE: no log
# 1 - BASE: basic log
# 2 - SERV: mincraft server log
# 3 - DEVE: developement log
# 4 - BYTE: connection bytes log
```
Allow the server to suspend server process when there are no players online  
- pro:  player wait time to join frozen server is ~0  
- cons: ram usage as minecraft server without msh (cpu remains ~0)  
_To mitigate ram usage you can set a high swappiness (on linux)_  
```yaml
"AllowSuspend": false
```
Hibernation and Starting server description
```yaml
"InfoHibernation": "                   §fserver status:\n                   §b§lHIBERNATING"
"InfoStarting": "                   §fserver status:\n                    §6§lWARMING UP"
```
Set to false if you don't want notifications (every 20 minutes)
```yaml
"NotifyUpdate": true
"NotifyMessage": true
```
Port to which players can connect
```yaml
"ListenPort": 25555
```
*30 seconds* is the time (after the last player disconnected) that msh waits before hibernating the minecraft server
```yaml
"TimeBeforeStoppingEmptyServer": 30	#any parameter more than 30s is recommended
```
Whitelist addresses or IPs that are allowed to start the server (leave empty to allow everyone)  
_unknown clients are not allowed to start the server, but can join_
```yaml
"Whitelist": ["127.0.0.1", "gekigek99"]
```

-----

### CREDITS:  

Author: [gekigek99](https://github.com/gekigek99)  

Contributors: [najtin](https://github.com/najtin/minecraft-server-hibernation), [f8ith](https://github.com/f8ith/minecraft-server-hibernation), [Br31zh](https://github.com/Br31zh/minecraft-server-hibernation), [someotherotherguy](https://github.com/someotherotherguy/minecraft-server-hibernation), [navidmafi](https://github.com/navidmafi), [cromefire](https://github.com/cromefire), [andreblanke](https://github.com/andreblanke), [KyleGospo](https://github.com/KyleGospo)  
Docker branch: [lubocode](https://github.com/lubocode/minecraft-server-hibernation)

_If you wish to contribute, please create a pull request using the dev branch as the base for your changes_

-----

<p align="center" >
    <a href="https://www.buymeacoffee.com/gekigek99" >
        <img src="https://raw.githubusercontent.com/gekware/minecraft-server-hibernation/c6a80ea835bea9f9a795c0805ab0e99ba326273c/res/icon/buymeacoffee.png" >
    </a>
</p>

<h4 align="center" >
    Give a star to this repository on <a href="https://github.com/gekware/minecraft-server-hibernation" > github</a>!
</h4>

<p align="center" >
    <a href="https://github.com/gekware/minecraft-server-hibernation/stargazers" >
        <img src="https://reporoster.com/stars/gekware/minecraft-server-hibernation" >
    </a>
</p>
