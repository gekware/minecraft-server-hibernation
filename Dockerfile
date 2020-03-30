FROM alpine AS firstStage
RUN apk add --no-cache git
# Clone repo with script change to subdir and compile
RUN git clone https://github.com/lubocode/minecraft-vanilla-server-hibernation.git


# Image for running script
FROM openjdk:8-jre-alpine
LABEL \
    maintainer="lubocode@outlook.com" \
    org.label-schema.name="minecraftserver-hibernation" \
    org.label-schema.description="OpenJDK image with Python script for automatically starting and stopping the supplied minecraft_server.jar" \
    org.label-schema.url="https://www.minecraft.net/download/server/" \
    org.label-schema.vcs-url="https://github.com/lubocode/minecraft-vanilla-server-hibernation"
RUN apk add --no-cache python3
# Create user without password or home driectory for running script and minecraftserver
RUN adduser -D -H runtimeuser
USER runtimeuser
# Expose Port specified in script
EXPOSE 25555
# Copy license, readme and python script from first stage
COPY --from=firstStage /minecraft-vanilla-server-hibernation/LICENSE /minecraftserver/
COPY --from=firstStage /minecraft-vanilla-server-hibernation/README.md /minecraftserver/
COPY --from=firstStage /minecraft-vanilla-server-hibernation/minecraft-vanilla-server-hibernation.py /minecraftserver/
# Volume to copy script into and for user to insert Minecraft server Java file
VOLUME ["/minecraftserver"]
ENTRYPOINT [ "/minecraftserver/minecraft-server-vanilla-hibernation.py" ]