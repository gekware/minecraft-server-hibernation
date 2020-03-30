# Image for compiling script
FROM golang:alpine AS buildstage
RUN apk add --no-cache git
# Clone repo with script, change to subdir and compile
RUN git clone https://github.com/lubocode/minecraft-vanilla-server-hibernation.git /usr/src/minecraft-vanilla-server-hibernation/
WORKDIR /usr/src/minecraft-vanilla-server-hibernation/go-version
RUN go build minecraft-vanilla-server-hibernation.go


# Image for running script
FROM openjdk:8-jre-alpine
LABEL \
    maintainer="lubocode@outlook.com" \
    org.label-schema.name="minecraftserver-hibernation" \
    org.label-schema.description="OpenJDK image with go script for automatically starting and stopping the supplied minecraft_server.jar" \
    org.label-schema.url="https://www.minecraft.net/download/server/" \
    org.label-schema.vcs-url="https://github.com/lubocode/minecraft-vanilla-server-hibernation/tree/master/go-version"
# Add screen utility to start and stop MC server
RUN apk add --no-cache screen
# Create user without password or home driectory for running script and minecraftserver
RUN adduser -D -H runtimeuser
USER runtimeuser
# Expose Port specified in script
EXPOSE 25555
# Copy compiled go script from first stage
COPY --from=buildstage /usr/src/minecraft-vanilla-server-hibernation/go-version/minecraft-vanilla-server-hibernation .
# Volume to copy script into and for user to insert Minecraft server Java file
VOLUME ["/minecraftserver"]
ENTRYPOINT [ "./minecraft-vanilla-server-hibernation" ]