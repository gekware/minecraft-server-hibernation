# minecraft-server-hibernation (Python)
## Docker (lubocode/minecraftserver-hibernate on Dockerhub) is built upon the Go script

version 4.2

concept, early-code and lastest improvements by [gekigek99](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)\
contributor (advanced-code) by [najtin](https://github.com/najtin/minecraft-server-hibernation)\
derived from [supernifty](https://github.com/supernifty/port-forwarder)\


## INSTRUCTIONS

This is a simple Python script to start a minecraft server on request and stop it when there are no player online.
How to use:

1. Install and run your desiered minecraft server
2. Rename the minecraft-server-jar to 'minecraft_server.jar'
3. Check the server-port parameter in 'server.properties': it should be 25565
4. Edit the paramters in the script as needed (you should modify START_MINECRAFT_SERVER, STOP_MINECRAFT_SERVER, MINECRAFT_SERVER_STARTUPTIME, TIME_BEFORE_STOPPING_EMPTY_SERVER )
5. run the script at reboot
6. you can connect to the server through port 25555

**IMPORTANT**
If you are the first to access to minecraft world you will *have to wait the specified amount of time in seconds* and then try to connect again.

```Python
MINECRAFT_SERVER_STARTUPTIME = 20       #any parameter more than 10s is recommended
```

After 120 seconds you have 120 to connect to the server before it is shutdown.

```Python
TIME_BEFORE_STOPPING_EMPTY_SERVER = 120 #any parameter more than 60s is recommended
```

You can change these parameters to fit your needs.


#### If you like what I do please consider having a cup of coffee with me at: https://www.buymeacoffee.com/gekigek99
