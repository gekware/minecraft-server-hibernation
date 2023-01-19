# Minecraft Server Hibernation  

[![msh - loc](https://tokei.rs/b1/github/gekware/minecraft-server-hibernation)](https://github.com/gekware/minecraft-server-hibernation)
[![msh - release](https://img.shields.io/github/release/gekware/minecraft-server-hibernation?color=05aefc)](https://github.com/gekware/minecraft-server-hibernation/releases)
[![msh - goreport](https://goreportcard.com/badge/github.com/gekware/minecraft-server-hibernation)](https://goreportcard.com/report/github.com/gekware/minecraft-server-hibernation)
[![msh - license](https://img.shields.io/github/license/gekware/minecraft-server-hibernation?color=6fff00)](https://github.com/gekware/minecraft-server-hibernation/blob/master/LICENSE)
[![msh - stars](https://img.shields.io/github/stars/gekware/minecraft-server-hibernation?color=ffbd19)](https://github.com/gekware/minecraft-server-hibernation/stargazers)

Avoid wasting resources by starting your Minecraft server automatically when a player joins and stopping it when no one is online  
_(for vanilla/modded on linux/windows/macos)_  

<p align="center" >
    <a href="https://github.com/gekware/minecraft-server-hibernation" >
        <img src="https://raw.githubusercontent.com/gekware/minecraft-server-hibernation/c6a80ea835bea9f9a795c0805ab0e99ba326273c/res/icon/msh.png" >
    </a>
</p>

version: v2.4.12  
Copyright (C) 2019-2022 [gekigek99](https://github.com/gekigek99)  

Check the [releases](https://github.com/gekware/minecraft-server-hibernation/releases) to download the binaries (for linux, windows and macos)

_You can compile msh from the dev branch to access a more recent version, but note that it may still need to be tested_

-----
### PROGRAM COMPILATION:
This version was successfully compiled in go version 1.19  
Compilation procedure:
```
git clone https://github.com/gekware/minecraft-server-hibernation.git  
cd minecraft-server-hibernation/  
go build .
```

-----
### INSTRUCTIONS:
1. Install the Minecraft server you want
2. Edit the parameters in `msh-config.json` as needed (*check definitions*):
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
7. You can connect to the server using the port from the configuration file (default 25555).

_\* = it's not compulsory to modify this parameter_

#### notes
- _`msh-config.json` is not generated automatically. You will need to download it from the [releases](https://github.com/gekware/minecraft-server-hibernation/releases)._
- _Automatically run msh at reboot._
- _In `server.properties` set `server-ip=0.0.0.0` to avoid errors when msh tries to connect to the minecraft server._
- _You must remove all braces from `msh-config.json`._  

-----
### DEFINITIONS:
- _Some of these parameters can be configured with command-line arguments (--help to know which)_  

Location of server folder and executable. You can find protocol/version [here](https://wiki.vg/Protocol_version_numbers) (but msh should set them automatically):
```yaml
"Server": {
  "Folder": "{path/to/server/folder}"
  "FileName": "{server.jar}"
  "Version": "1.19.2"
  "Protocol": 760
}
```

Commands to start and stop minecraft server  
_StopServerAllowKill allows to kill the server after a certain amount of time (in seconds) when it's not responding_
```yaml
"Commands": {
  "StartServer": "java <Commands.StartServerParam> -jar <Server.FileName> nogui"
  "StartServerParam": "-Xmx1024M -Xms1024M"
  "StopServer": "stop"
  "StopServerAllowKill": 10	# set to -1 to disable
}
```

Set the logging level for debug purposes
```yaml
"Debug": 1
# 0 - NONE: no log
# 1 - BASE: basic log
# 2 - SERV: minecraft server log
# 3 - DEVE: developement log
# 4 - BYTE: connection bytes log
```

Ports configuration
- _MshPort and MshPortQuery must be different from the respective ones in `server.properties`_
- _msh enables query handling if `enable-query=true` in `server.properties`_
```yaml
"MshPort": 25555		# port to which players can join
"MshPortQuery": 25555	# port to which stats query requests are performed 
```

TimeBeforeStoppingEmptyServer sets the time (after the last player disconnected) that msh waits before hibernating the minecraft server
```yaml
"TimeBeforeStoppingEmptyServer": 30
```

SuspendAllow enables msh to suspend minecraft server process when there are no players online  
_To mitigate ram usage you can set a high swappiness (on linux)_  
- pro:  player wait time to join frozen server is ~0  
- cons: ram usage as minecraft server without msh (cpu remains ~0)  

SuspendRefresh enables refresh of minecraft server suspension every set seconds (to avoid watchdog crash at unsuspension)  
- setting `these variables` and `SuspendRefresh` might prevent minecraft server watchdog crash when `SuspendAllow` is enabled  

|       file        |                       variable                       |
| ----------------- | ---------------------------------------------------- |
| server.properties | `max-tick-time= -1`                                  |
| spigot.yml        | `timeout-time: -1`, `restart-on-crash: false`        |
| bukkit.yml        | `warn-on-overload: false`                            |
| paper-global.yml  | `early-warning-delay: -1`, `early-warning-every: -1` |

```yaml
"SuspendAllow": false
"SuspendRefresh": -1	# set -1 to disable, advised value: 120 (reduce if minecraft server keeps crashing)
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

Whitelist contains IPs and player names that are allowed to start the server (leave empty to allow everyone)  
WhitelistImport adds `whitelist.json` to player names that are allowed to start the server  
_unknown clients are not allowed to start the server, but can join_  
```yaml
"Whitelist": ["127.0.0.1", "gekigek99"]
"WhitelistImport": false
```

-----

### CREDITS:  

Author: [gekigek99](https://github.com/gekigek99)  

Contributors: [najtin](https://github.com/najtin), [f8ith](https://github.com/f8ith), [Br31zh](https://github.com/Br31zh), [someotherotherguy](https://github.com/someotherotherguy), [navidmafi](https://github.com/navidmafi), [cromefire](https://github.com/cromefire), [andreblanke](https://github.com/andreblanke), [KyleGospo](https://github.com/KyleGospo)  

Docker branch (outdated): [lubocode](https://github.com/lubocode/minecraft-server-hibernation)  

Pterodactyl egg: [BolverBlitz](https://github.com/gekware/minecraft-server-hibernation-pterodactyl-egg)  

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
