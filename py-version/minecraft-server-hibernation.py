#!/usr/bin/env python3

import socket
import os, sys, platform
from threading import Thread, Timer, Lock
from subprocess import Popen, PIPE, STDOUT 
import logging
import math
import json
import atexit

## contains intro to script and program
info = [
	"Minecraft-Server-Hibernation is used to auto-start/stop a vanilla/modded minecraft server",
	"Copyright (C) 2019-2020 gekigek99",
	"v6.6 (Python)",
	"visit my github page: https://github.com/gekigek99",
	"If you like what I do please consider having a cup of coffee with me at: https://www.buymeacoffee.com/gekigek99"
]

##------------------------don't modify------------------------##

# loads config.json into config
with open("config.json", "r") as f:
	config = json.load(f)

# to keep track of the minecraft server status ("offline", "starting", "online")
serverStatus = "offline"

# to keep track of players connected to the server
players = 0

# to calculate the bytes/s from/to server
dataCountBytesToClients, dataCountBytesToServer = 0, 0

# to keep track of how many times stopEmptyMinecraftServer() has been called in the last {TimeBeforeStoppingEmptyServer} seconds
stopInstances = 0

# to keep track of how many seconds are still needed to reach serverStatus == "online"
timeLeftUntilUp = config["tomodify"]["minecraftServerStartupTime"]
lock = Lock()

##--------------------------PROGRAM---------------------------##

def startMinecraftServer():
	global serverStatus, players, timeLeftUntilUp

	serverStatus = "starting"
	
	# block that execute the correct start command depending on the OS
	if platform.system() == "Linux":
		os.system(config["tomodify"]["startMinecraftServerLin"])
	elif platform.system() == "Windows":
		startMinecraftServer.mineServTerminal = Popen(config["tomodify"]["startMinecraftServerWin"].split(), stdout=PIPE, stdin=PIPE, stderr=STDOUT)
	else:
		logging.info("OS not supported!")
		sys.exit(0)
	
	logging.info("MINECRAFT SERVER IS STARTING!")

	# initialization of players
	players = 0
	
	## sets serverStatus == "online"
	## increases stopInstances by one. after {TimeBeforeStoppingEmptyServer} executes stopEmptyMinecraftServer(false)
	def setServerStatusOnline():
		global serverStatus, stopInstances, lock
		serverStatus = "online"
		logging.info("MINECRAFT SERVER IS UP!")
		with lock:
			stopInstances += 1
		Timer(config["tomodify"]["timeBeforeStoppingEmptyServer"], stopEmptyMinecraftServer, ()).start()
	# updates timeLeftUntilUp each second. if timeLeftUntilUp == 0 it executes setServerStatusOnline()
	def updatetTimeLeft():
		global timeLeftUntilUp
		if timeLeftUntilUp > 0:
			timeLeftUntilUp -= 1
			Timer(1, updatetTimeLeft, ()).start()
		elif timeLeftUntilUp == 0 :
			setServerStatusOnline()

	updatetTimeLeft()

def stopEmptyMinecraftServer(forceExec=False):
	global serverStatus, timeLeftUntilUp, stopInstances, lock
	
	if forceExec and serverStatus != "offline":
		# skip some checks to issue the stop server command forcefully
		pass
	else:
		# check that there is only one "stop server command" instance running and players <= 0 and serverStatus != "offline".
		# on the contrary the server won't be stopped
		with lock:
			stopInstances -= 1
			if stopInstances > 0 or players > 0 or serverStatus == "offline":
				return
	
	serverStatus = "offline"
	
	# block that execute the correct stop command depending on the OS
	if platform.system() == "Linux":
		os.system(config["tomodify"]["stopMinecraftServerLin"])
	elif platform.system() == "Windows":
		startMinecraftServer.mineServTerminal.communicate(input=config["tomodify"]["stopMinecraftServerWin"].encode())[0]
	else:
		logging.info("OS not supported!")
		sys.exit(0)
	
	if forceExec:
		logging.info("MINECRAFT SERVER IS FORCEFULLY SHUTTING DOWN!")
	else:
		logging.info("MINECRAFT SERVER IS SHUTTING DOWN!")
	
	# reset timeLeftUntilUp to initial value
	timeLeftUntilUp = config["tomodify"]["minecraftServerStartupTime"]

def printDataUsage():
	global dataCountBytesToClients, dataCountBytesToServer, lock
	with lock:
		if dataCountBytesToClients != 0 or dataCountBytesToServer != 0:
			logger("data/s: {:8.3f} KB/s to clients | {:8.3f} KB/s to server".format(dataCountBytesToClients/1024, dataCountBytesToServer/1024))
			dataCountBytesToClients = 0
			dataCountBytesToServer = 0
	Timer(1, printDataUsage).start()

def main():
	# prints intro to program
	print("\n".join(info[1:4]))

	# launch printDataUsage()
	Thread(target=printDataUsage).start()

	# open a listener on {config.Advanced.ListenHost}+":"+{config.Advanced.ListenPort}
	listener = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
	listener.setblocking(1)
	listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)   #to prevent errno 98 address already in use
	listener.bind((config["advanced"]["listenHost"], int(config["advanced"]["listenPort"])))
	listener.listen(5)

	logging.info("*** listening for new clients to connect...")
	
	# infinite cycle to accept clients. when a clients connects it is passed to handleClientSocket()
	while True:
		try:
			clientSocket, clientAddress = listener.accept()        #blocking
			handleClientSocket(clientSocket, clientAddress)
		except Exception as e:
			logger("Exception in main():", str(e))

# to handle a client that is connecting.
# can handle a client that is requesting server info or trying to join.
def handleClientSocket(clientSocket, clientAddress):
	try:
		logger("*** from {}:{} to {}:{}".format(clientAddress[0], config["advanced"]["listenPort"], config["advanced"]["targetHost"], config["advanced"]["targetPort"]))
		
		# block containing the case of serverStatus == "offline" or "starting"
		if serverStatus == "offline" or serverStatus == "starting":
			# read first packet
			buffer = clientSocket.recv(1024)
						
			# \x00 or \x01 is the last byte of the first packet when requesting server info
			# \xd3 as last byte is used when sending the first packet for the second time (after unsuccessful first packet reception)
			# \xd3 could also be used when the player is trying to join!
			if buffer[-1] == 0 or buffer[-1] == 1:
				if serverStatus == "offline":
					logging.info("player unknown requested server info from "+str(clientAddress[0]))
					# answer to client with emulated server info
					clientSocket.sendall(buildMessage("info", "                   &fserver status:\n                   &b&lHIBERNATING"))
		
				elif serverStatus == "starting":
					logging.info("player unknown requested server info from "+str(clientAddress[0])+" during server startup")
					# answer to client with emulated server info
					clientSocket.sendall(buildMessage("info", "                   &fserver status:\n                    &6&lWARMING UP"))
				
				# answer to client with ping
				answerPingReq(clientSocket)
			
			# the client first message is {data, \x02} --> the client is trying to join the server
			if buffer[-1] == 2:
				# read second packet (contains the playerName)
				buffer = clientSocket.recv(1024)
				playerName = buffer[3:].decode(errors="replace")
				
				if serverStatus == "offline":
					# client is trying to join the server and serverStatus == "offline" --> issue startMinecraftServer()
					startMinecraftServer()
					logging.info(playerName + " tried to join from " + clientAddress[0])
					# answer to client with text in the loadscreen
					clientSocket.sendall(buildMessage("txt", "Server start command issued. Please wait... Time left: " + str(timeLeftUntilUp) + " seconds"))
				
				elif serverStatus == "starting":
					logging.info(playerName + " tried to join from " + clientAddress[0] + " during server startup")
					clientSocket.sendall(buildMessage("txt", "Server is starting. Please wait... Time left: " + str(timeLeftUntilUp) + " seconds"))

			# since the server is still not online, close the client connection
			logger("closing connection for:", clientAddress[0])
			clientSocket.shutdown(1)   #sends FIN to client
			clientSocket.close()

		# block containing the case of serverStatus == "online"
		if serverStatus == "online":
			# if the server is online, just open a connection with the server and connect it with the client
			serverSocket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
			serverSocket.connect((config["advanced"]["targetHost"], int(config["advanced"]["targetPort"])))

			connectSocketsAsync(clientSocket, serverSocket)

	except Exception as e:
		logger("Exception in handleClientSocket():", str(e))

# launches clientToServer() and serverToClient()
def connectSocketsAsync(client, server):
	Thread(target=clientToServer, args=(client, server, )).start()
	Thread(target=serverToClient, args=(server, client, )).start()

def clientToServer(source, destination):
	global players, stopInstances, lock
	players +=1
	logging.info("A PLAYER JOINED THE SERVER! - "+str(players)+" players online")

	# exchanges data from client to server (isServerToClient == false)
	forwardSync(source, destination, False)

	players -= 1
	logging.info("A PLAYER LEFT THE SERVER! - "+str(players)+" players remaining")

	# this block increases stopInstances by one and starts the timer to execute stopEmptyMinecraftServer(false)
	# (that will do nothing in case there are players online)
	with lock:
		stopInstances += 1

	Timer(config["tomodify"]["timeBeforeStoppingEmptyServer"], stopEmptyMinecraftServer, ()).start()

def serverToClient(source, destination):
	# exchanges data from server to client (isServerToClient == true)
	forwardSync(source, destination, True)

# forwardSync takes a source and a destination net.Conn and forwards them.
# (isServerToClient used to know the forward direction)
def forwardSync(source, destination, isServerToClient):
	global dataCountBytesToServer, dataCountBytesToClients, lock, config
	
	data = b" "

	# set to false after the first for cycle
	firstBuffer = True
	
	# set read and write timeout
	source.settimeout(config["tomodify"]["timeBeforeStoppingEmptyServer"])
	destination.settimeout(config["tomodify"]["timeBeforeStoppingEmptyServer"])
	
	try:
		while True:
			# read data from source
			data = source.recv(1024)
			# if there is no data close the source
			if not data:
				source.shutdown(1)
				source.close()
				break

			# write data to destination
			destination.sendall(data)

			# if debug == true --> calculate bytes/s to client/server
			if config["advanced"]["debug"]:
				with lock:
					if isServerToClient:
						dataCountBytesToClients = dataCountBytesToClients + len(data)
					else:
						dataCountBytesToServer = dataCountBytesToServer + len(data)
            
			# this block is used to find the serverVersion and serverProtocol.
			# these parameters are only found in serverToClient connection in the first buffer that is read
			# if the above specified buffer contains "\"version\":{\"name\":\"" and ",\"protocol\":" --> extract the serverVersion and serverProtocol
			if isServerToClient and firstBuffer and b"\"version\":{\"name\":\"" in data and b",\"protocol\":" in data:
				newServerVersion = str(data).split("\"version\":{\"name\":\"")[1].split("\",")[0]
				newServerProtocol = str(data).split(",\"protocol\":")[1].split("}")[0]

				# if serverVersion or serverProtocol are different from the ones specified in config.json --> update them
				if newServerVersion != config["advanced"]["serverVersion"] or newServerProtocol != config["advanced"]["serverProtocol"]:
					config["advanced"]["serverVersion"] = newServerVersion
					config["advanced"]["serverProtocol"] = newServerProtocol

					logger(
						"new server version or protocol found!",
						"serverVersion:", config["advanced"]["serverVersion"],
						"serverProtocol:", config["advanced"]["serverProtocol"]
					)

					try:
						with open("config.json", "w") as f:
							# write the struct config to config.json
							json.dump(config, f, indent=2)
							logger("saved to config.json")
					except:
						logger("could not update config.json")
            # first cycle is finished, set firstBuffer = false
			firstBuffer = False
	
	except IOError as e:
		if e.errno == 32:               #user/server disconnected normally. has to be catched, because there is a race condition
			return                      #when trying to check if destination.recv does return data
		logger("IOError in forward():", str(e))
	except Exception as e:
		logger("Exception in forward():", str(e))

##---------------------------utils----------------------------##

# takes the format ("txt", "info") and a message to write to the client
def buildMessage(format, message):
	def mountHeader(message, const):
		# mountHeader: mounts the header to a specified message
		# scheme: 			[header1 												|header2 	|header3 					|message]
		# bytes used:		[1/2													|1			|1/2						|∞		]
		# value calc: 		[len(header2) + len(header3) + len(message) + constant	|0			|len(message) + constant 	|message]
		# possible values:	[1            + 1/2          + ∞            + 0/11264	|0			|∞            + 0/11264		|-------]
		
		def addHeader(message):
			mesLen = len(message) + const
			# calculate the bytes needed to store mesLen
			# int(math.Ceil(log255(mesLen)))
			byteNum = math.ceil(math.log(mesLen, 255))
			return (mesLen).to_bytes(byteNum, byteorder="little") + message

		messageByte = message.encode()

		# header3 calculation
		messageByte = addHeader(messageByte)
		
		# header2 calculation
		messageByte = bytes([0]) + messageByte

		# header1 calculation
		messageByte = addHeader(messageByte)

		return messageByte

	if format == "txt":
		# to display text in the loadscreen

		messageJSON = "".join((
			"{",
			"\"text\":\"", message, "\"",
			"}"))

		# for txt the constant == 0
		messageHeader = mountHeader(messageJSON, 0)

	elif format == "info":
		# to send server info

		# captured examples:
		# \xf8W\x00\xf5W{
		# "description":{"text":"\xc2\xa7fServer status:\xc2\xa7r\\n                  \xc2\xa7b\xc2\xa7l\xc2\xa7oHIBERNATING"},
		# "players":{"max":20,"online":0},
		# "version":{"name":"1.16.2","protocol":751},
		# "favicon":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAYAAACqaXHeAAAgK0lEQ ... mvCAAAAAElFTkSuQmCC
		# }

		# [239][26][0][236][26]{
		# "description":{"text":"\xc2\xa7fServer status:\xc2\xa7r\\n                  \xc2\xa7b\xc2\xa7l\xc2\xa7oHIBERNATING"},
		# "players":{"max":20,"online":1,"sample":[{"id":"bf043252-d4a5-3165-b37e-d2b4c1984683","name":"gekigek99"}]},
		# "version":{"name":"1.16.3","protocol":753},
		# "favicon":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAYAAACqaXHeAAAJE0lEQ ... mvCAAAAAElFTkSuQmCC
		# }

		# in message: "\n" -> "&r\\n" then "&" -> "\xa7"
		# in py only \xa7 is replaced since the python .encode() metod (in mountHeader) adds the \xc2 byte
		messageAdapted = message.replace("\n", "&r\\n").replace("&", "\xa7")

		messageJSON = "".join(("{",
			"\"description\":{\"text\":\"", messageAdapted, "\"},",
			"\"version\":{\"name\":\"", config["advanced"]["serverVersion"], "\",\"protocol\":", str(config["advanced"]["serverProtocol"]), "},",
			"\"favicon\":\"", serverIcon, "\"",
			"}"))
		
		# for info the constant == 11264
		messageHeader = mountHeader(messageJSON, 11264)

	else:
		logger("buildMessage: specified format invalid")
		messageHeader = None

	return messageHeader

def answerPingReq(clientSocket):
	# read the first packet
	req = clientSocket.recv(1024)
	
	# if req == b"\x01\x00" --> read again (the correct ping byte array have still to arrive)
	if req == b"\x01\x00":
		req = clientSocket.recv(1024)
	
	# answer the ping request
	clientSocket.sendall(req)

# prints the args if debug option is set to true
def logger(*args):
	if config["advanced"]["debug"]:
		logging.info(' '.join(args))

##------------------------py specific-------------------------##

# specify default parameters for logging
logging.basicConfig(
	level=logging.INFO,
	format="%(asctime)s %(message)s",
	datefmt="%d-%b-%y %H:%M:%S"
)

# listen for interrupt signal and issue stopEmptyMinecraftServer(true) before exiting
atexit.register(stopEmptyMinecraftServer, forceExec=True)

##---------------------------data-----------------------------##

# contains is the captured picture data of the msh logo
serverIcon = (
	"data:image/png;base64,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"
)

if __name__ == "__main__":
	main()
