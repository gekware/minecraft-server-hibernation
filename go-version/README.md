# README

Go script written and translated by [gekigek99](https://github.com/gekigek99/minecraft-vanilla-server-hibernation/tree/master/go-version)\
Slightly modified for docker usage by [lubocode](https://github.com/lubocode/minecraft-vanilla-server-hibernation/tree/master/go-version)

This image does **NOT** contain a minecraft server installation.\
Please insert your minecraft server files into the associated volume.
Your minecraft server file should lie in the top level of the volume and should be named minecraft_server.jar\
If you want to deviate from this, use the arguments specified below.
Similarly, if you want to change the amount of RAM for your MC server, have a look at the arguments as well.

The exposed container port is 25555. The script passes traffic through to 25565, which is MCs standard port.

**Usage:**

```bash
docker run \
    -p 25555:25555 \
    -v /docker/appdata/minecraftserver-hibernate:/minecraftserver:rw \
    -e minRAM=512M \
    -e maxRAM=2G \
    -e mcPath=/minecraftserver/ \
    -e mcFile=minecraft_server.jar \
    minecraftserver-hibernate
```
