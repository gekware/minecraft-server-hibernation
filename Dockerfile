FROM alpine AS firstStage
RUN apk add git
WORKDIR /usr/src/
# Clone repo with script change to subdir and compile
RUN git clone https://github.com/lubocode/minecraft-vanilla-server-hibernation.git


# Image for running script
FROM openjdk:8-jre-alpine
LABEL maintainer="lubocode@outlook.com"
RUN apk add python3
# Create user without password or home driectory for running script and minecraftserver
RUN adduser -D -H runtimeuser
USER runtimeuser
# Expose Port specified in script
EXPOSE 25555
# Copy license, readme and python script from first stage
COPY --from=firstStage /usr/src/minecraft-vanilla-server-hibernation/LICENSE /minecraftserver/LICENSE
COPY --from=firstStage /usr/src/minecraft-vanilla-server-hibernation/README.md /minecraftserver/README.md
COPY --from=firstStage /usr/src/minecraft-vanilla-server-hibernation/minecraft-vanilla-server-hibernation.py /minecraftserver/minecraft-server-hibernation.py
# Volume to copy script into and for user to insert Minecraft server Java file
VOLUME ["/minecraftserver"]
ENTRYPOINT [ "/minecraftserver/minecraft-server-hibernation.py" ]