# Image for compiling script
FROM golang:alpine AS buildstage
# Copy script, change to subdir and compile
COPY minecraft-vanilla-server-hibernation.go /usr/src/
WORKDIR /usr/src/
RUN go build minecraft-vanilla-server-hibernation.go

# Image for running script
FROM openjdk:8-jre-alpine
LABEL \
    maintainer="lubocode@outlook.com" \
    org.label-schema.name="minecraftserver-hibernation" \
    org.label-schema.description="OpenJDK image with go script for automatically starting and stopping the supplied Minecraft server." \
    org.label-schema.url="https://www.minecraft.net/download/server/" \
    org.label-schema.vcs-url="https://github.com/lubocode/minecraft-vanilla-server-hibernation/tree/master/go-version"
# Add bash and screen utility to start and stop MC server
RUN apk add --no-cache bash screen
# Create user without password or home driectory for running script and minecraftserver
RUN adduser -D -H runtimeuser
USER runtimeuser
# Expose Port specified in script
EXPOSE 25555
# Volume to copy script into and for user to insert Minecraft server Java file
VOLUME ["/minecraftserver"]
# Environment variables to change script parameters at runtime
ENV minRAM=512M \
    maxRAM=2G \
    mcPath=/minecraftserver/ \
    mcFile=minecraft_server.jar
# Copy compiled go script from first stage
COPY --from=buildstage /usr/src/minecraft-vanilla-server-hibernation .
ENTRYPOINT ./minecraft-vanilla-server-hibernation -minRAM=${minRAM} -maxRAM=${maxRAM} -mcPath=${mcPath} -mcFile=${mcFile}