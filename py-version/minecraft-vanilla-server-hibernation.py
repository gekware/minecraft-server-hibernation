#!/usr/bin/env python3

import socket
import os, sys
from threading import Thread, Timer, Lock
from time import sleep
from subprocess import Popen, PIPE, STDOUT
import platform
import logging

info = [
	"Minecraft-Vanilla-Server-Hibernation is used to auto-start/stop a vanilla minecraft server",
	"Copyright (C) 2019-2020 gekigek99",
	"v4.5 (Python)",
	"visit my github page: https://github.com/gekigek99",
	"If you like what I do please consider having a cup of coffee with me at: https://www.buymeacoffee.com/gekigek99"
]

##---------------------------modify---------------------------##

startMinecraftServerLin = "sudo systemctl start minecraft-server"    #set command to start minecraft-server service
stopMinecraftServerLin= "sudo systemctl stop minecraft-server"      #set command to stop minecraft-server service
startMinecraftServerWin = "java -Xmx1024M -Xms1024M -jar server.jar nogui"
stopMinecraftServerWin = "stop"

minecraftServerStartupTime = 20       #time the server needs until it is fully started
timeBeforeStoppingEmptyServer = 60  #time the server waits for clients to connect then it issues the stop command to server

##--------------------------advanced--------------------------##

listenHost = "0.0.0.0"
listenPort = 25555         #the port you will connect to on minecraft client

targetHost = "ribericloud-turin.duckdns.org"
targetPort = 25555         #the port specified on server.properties

debug = True               #if true more additional information is printed

##------------------------don't modify------------------------##

players = 0
dataCountBytesToServer, dataCountBytesToClients = 0, 0
serverStatus = "offline"
timeLeftUntilTp = minecraftServerStartupTime
stopInstances = 0
lock = Lock()

##------------------------py specific-------------------------##

##                            ...

logging.basicConfig(
	level=logging.INFO,
    format='%(asctime)s %(message)s',
	datefmt='%d-%b-%y %H:%M:%S'
)

def startMinecraftServer():
	global serverStatus, players, timeLeftUntilTp
	if serverStatus != "offline":
		return
	serverStatus = "starting"
	
	if platform.system() == "Linux":
		os.system(startMinecraftServerLin)
	elif platform.system() == "Windows":
		startMinecraftServer.mineServTerminal = Popen(startMinecraftServerWin.split(), stdout=PIPE, stdin=PIPE, stderr=STDOUT)
	else:
		logging.info("OS not supported!")
		sys.exit(0)
	
	logging.info("MINECRAFT SERVER IS STARTING!")
	players = 0
	def setServerStatusOnline():
		global serverStatus, stopInstances, lock
		serverStatus = "online"
		logging.info("MINECRAFT SERVER IS UP!")
		with lock:
			stopInstances += 1
		Timer(timeBeforeStoppingEmptyServer, stopEmptyMinecraftServer, ()).start()
	def updatetTimeLeft():
		global timeLeftUntilTp
		if timeLeftUntilTp > 0:
			timeLeftUntilTp-=1
			Timer(1, updatetTimeLeft, ()).start()
	updatetTimeLeft()
	Timer(minecraftServerStartupTime, setServerStatusOnline, ()).start()

def stopEmptyMinecraftServer():
	global serverStatus, timeLeftUntilTp, stopInstances, lock
	with lock:
		stopInstances -= 1
		if stopInstances > 0 or players > 0 or serverStatus == "offline":
			return
	serverStatus = "offline"
	
	if platform.system() == "Linux":
		os.system(stopMinecraftServerLin)
	elif platform.system() == "Windows":
		startMinecraftServer.mineServTerminal.communicate(input=stopMinecraftServerWin.encode())[0]
	else:
		logging.info("OS not supported!")
		sys.exit(0)
	
	logging.info("MINECRAFT SERVER IS SHUTTING DOWN!")
	timeLeftUntilTp = minecraftServerStartupTime

def printDataUsage():
	global dataCountBytesToServer, dataCountBytesToClients, lock
	with lock:
		if dataCountBytesToServer != 0 or dataCountBytesToClients != 0:
			logger("data/s: {:8.3f} KB/s to clients | {:8.3f} KB/s to server\n".format(dataCountBytesToClients/1024, dataCountBytesToServer/1024))
			dataCountBytesToServer = 0
			dataCountBytesToClients = 0
	Timer(1, printDataUsage, ()).start()

def main():
	print("\n".join(info[1:4]))
	dockSocket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	dockSocket.setblocking(1)
	dockSocket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)   #to prevent errno 98 address already in use
	dockSocket.bind((listenHost, listenPort))
	dockSocket.listen(5)
	logging.info("*** listening for new clients to connect...")
	printDataUsage()
	while True:
		try:
			clientSocket, clientAddress = dockSocket.accept()        #blocking
			Thread(target=handleClientSocket, args=(clientSocket, clientAddress, )).start()
		except Exception as e:
			logger("Exception in main(): "+str(e))

def handleClientSocket(clientSocket, clientAddress):
	try:
		logger("*** from {}:{} to {}:{}".format(clientAddress[0], listenPort, targetHost, targetPort))
		if serverStatus == "offline" or serverStatus == "starting":
			buffer = clientSocket.recv(1024)
		
			if buffer[-1] == 1:   #\x01 is the last byte of the first message when requesting server info
				if serverStatus == "offline":
					logging.info("player unknown requested server info from "+str(clientAddress[0]))
				elif serverStatus == "starting":
					logging.info("player unknown requested server info from "+str(clientAddress[0])+" during server startup")
			
			elif buffer[-1] == 2:                   #\x02 is the last byte of the first message when player is trying to join the server
				buffer = clientSocket.recv(1024)    #here it"s reading an other packet containing the player name
				playerName = buffer[3:].decode(errors="replace")
				
				if serverStatus == "offline":
					startMinecraftServer()
					logging.info(playerName, "tryed to join from", clientAddress[0])
					clientSocket.sendall(BuildMessage("Server start command issued. Please wait... Time left: " + str(timeLeftUntilTp) + " seconds"))
				elif serverStatus == "starting":
					logging.info(playerName, "tryed to join from", clientAddress[0], "during server startup")
					clientSocket.sendall(BuildMessage("Server is starting. Please wait... Time left: " + str(timeLeftUntilTp) + " seconds"))
			
			logger("closing connection for: "+clientAddress[0])
			clientSocket.shutdown(1)   #sends FIN to client
			clientSocket.close()

		if serverStatus == "online":    
			serverSocket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
			serverSocket.connect((targetHost, targetPort))

			connectSocketsAsync(clientSocket, serverSocket)

	except Exception as e:
		logger("Exception in handleClientSocket(): "+str(e))

def connectSocketsAsync(client, server):
	Thread(target=clientToServer, args=(client, server, )).start()
	Thread(target=serverToClient, args=(server, client, )).start()

def clientToServer(source, destination):
	global players, stopInstances, lock
	players +=1
	logging.info("A PLAYER JOINED THE SERVER! - "+str(players)+" players online")

	forwardSync(source, destination, False)

	players -= 1
	logging.info("A PLAYER LEFT THE SERVER! - "+str(players)+" players remaining")

	with lock:
		stopInstances += 1

	Timer(timeBeforeStoppingEmptyServer, stopEmptyMinecraftServer, ()).start()

def serverToClient(source, destination):
	forwardSync(source, destination, True)

def forwardSync(source, destination, isServerToClient):
	global dataCountBytesToServer, dataCountBytesToClients, lock
	data = b" "
	source.settimeout(timeBeforeStoppingEmptyServer)
	destination.settimeout(timeBeforeStoppingEmptyServer)

	try:
		while True:
			data = source.recv(1024)
			if not data:                #if there is no data stop listening, this means the socket is closed
				break
			destination.sendall(data)

			if debug:
				with lock:
					if isServerToClient:
						dataCountBytesToClients = dataCountBytesToClients + len(data)
					else:
						dataCountBytesToServer = dataCountBytesToServer + len(data)
	
	except IOError as e:
		if e.errno == 32:               #user/server disconnected normally. has to be catched, because there is a race condition
			return                      #when trying to check if destination.recv does return data
		logger("IOError in forward(): " + str(e))
	except Exception as e:
		logger("Exception in forward(): " + str(e))

##---------------------------utils----------------------------##

def BuildMessage(message):
	message = "{\"text\":\"" + message + "\"}"
	message = bytes([len(message) + 2]) + bytes([0]) + bytes([len(message)]) + message.encode()
	return message

def logger(message):
	if debug:
		logging.debug(message, format="%(asctime)s - %(message)s")

if __name__ == "__main__":
	main()
