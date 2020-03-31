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
COPY LICENSE /minecraftserver/
COPY README.md /minecraftserver/
COPY minecraft-vanilla-server-hibernation.py /minecraftserver/
# Volume to copy script into and for user to insert Minecraft server Java file
VOLUME ["/minecraftserver"]
ENTRYPOINT [ "/minecraftserver/minecraft-server-vanilla-hibernation.py" ]