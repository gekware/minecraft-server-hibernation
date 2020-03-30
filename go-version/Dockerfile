# Image for compiling script
FROM golang:alpine AS buildstage
RUN apk add --no-cache git
# Clone repo with script, change to subdir and compile
RUN git clone https://github.com/lubocode/minecraft-vanilla-server-hibernation.git
WORKDIR /minecraft-vanilla-server-hibernation/go-version/
RUN go build minecraft-vanilla-server-hibernation.go


# Image for running script
FROM openjdk:8-jre-alpine
LABEL \
    maintainer="lubocode@outlook.com" \
    org.label-schema.name="minecraftserver-hibernation" \
    org.label-schema.description="OpenJDK image with go script for automatically starting and stopping the supplied minecraft_server.jar" \
    org.label-schema.url="https://www.minecraft.net/download/server/" \
    org.label-schema.vcs-url="https://github.com/lubocode/minecraft-vanilla-server-hibernation/tree/master/go-version"
# Create user without password or home driectory for running script and minecraftserver
RUN adduser -D -H runtimeuser
USER runtimeuser
# Expose Port specified in script
EXPOSE 25555
# Copy compiled go script from first stage
COPY --from=buildstage /minecraft-vanilla-server-hibernation/go-version/minecraft-vanilla-server-hibernation .
# Volume to copy script into and for user to insert Minecraft server Java file
VOLUME ["/minecraftserver"]
# Arguments for RAM size and Minecraft path
ARG minRAM="512M"
ARG maxRAM="2G"
ARG minecraftPath="/minecraftserver/"
ARG minecraftFileName="minecraft_server.jar"
ENV Xms=${minRAM} Xmx=${maxRAM} mcPath=${minecraftPath} mcFile=${minecraftFileName}
ENTRYPOINT ./minecraft-vanilla-server-hibernation -minRAM "-Xms$Xms" -maxRAM "-Xmx$Xmx" -mcPath "$mcPath" -mcFile "$mcFile"