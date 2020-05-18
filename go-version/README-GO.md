# minecraft-vanilla-server-hibernation v1.1 (Go)

#### If you like what I do please consider having a cup of coffee with me at: https://www.buymeacoffee.com/gekigek99

version 1.1 (Go)

written and translated by [gekigek99](https://github.com/gekigek99/minecraft-vanilla-server-hibernation)
slightly modified for docker usage by [lubocode](https://github.com/lubocode/minecraft-vanilla-server-hibernation)


## Compile program:
This version was successfully compiled in go-version 1.14

To compile, run the command:

```bash
go build minecraft-vanilla-server-hibernation.go
```

The same python-version setup configuration applyes to this version

For specifying minimum and maximum amount of RAM and the Minecraft path use the compiled file as follows:

```bash
./minecraft-vanilla-server-hibernation -minRAM "-Xms512M" -maxRAM "-Xmx2G" -mcPath "/minecraftserver/" -mcFile "minecraft_server.jar"
```

#### If you like what I do please consider having a cup of coffee with me at: https://www.buymeacoffee.com/gekigek99
