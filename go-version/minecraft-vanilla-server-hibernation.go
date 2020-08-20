package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

var info []string = []string{
	"Minecraft-Vanilla-Server-Hibernation is used to auto-start/stop a vanilla minecraft server",
	"Copyright (C) 2019-2020 gekigek99",
	"v1.4 (Go)",
	"visit my github page: https://github.com/gekigek99",
	"if you like what I do please consider having a cup of coffee with me at: https://www.buymeacoffee.com/gekigek99",
}

//---------------------------modify---------------------------//

const startMinecraftServerLin = "systemctl start minecraft-server"
const stopMinecraftServerLin = "systemctl stop minecraft-server"
const startMinecraftServerWin = "java -Xmx1024M -Xms1024M -jar server.jar nogui"
const stopMinecraftServerWin = "stop"

const minecraftServerStartupTime = 20
const timeBeforeStoppingEmptyServer = 60

//--------------------------advanced--------------------------//

const listenHost = "0.0.0.0"
const listenPort = "25555"

const targetHost = "127.0.0.1"
const targetPort = "25565"

const debug = false

//------------------------don't modify------------------------//

var players int = 0
var dataCountBytesToServer, dataCountBytesToClients float64 = 0, 0
var serverStatus string = "offline"
var timeLeftUntilTp int = minecraftServerStartupTime
var stopInstances int = 0
var mutex = &sync.Mutex{}

//------------------------go specific-------------------------//

var cmdIn io.WriteCloser

//--------------------------PROGRAM---------------------------//

func startMinecraftServer() {
	if serverStatus != "offline" {
		return
	}
	serverStatus = "starting"

	if runtime.GOOS == "linux" {
		err := exec.Command("/bin/bash", "-c", startMinecraftServerLin).Run()
		if err != nil {
			log.Printf("error starting minecraft server: %v\n", err)
		}
	} else if runtime.GOOS == "windows" {
		cmd := exec.Command(strings.Split(startMinecraftServerWin, " ")[0], strings.Split(startMinecraftServerWin, " ")[1:]...)
		cmdIn, _ = cmd.StdinPipe()
		cmd.Start()
	} else {
		log.Print("error: OS not supported!")
		os.Exit(1)
	}

	log.Print("*** MINECRAFT SERVER IS STARTING!")
	players = 0
	updateTimeleft()
	go timer(minecraftServerStartupTime, setServerStatusOnline)
}

func stopEmptyMinecraftServer() {
	mutex.Lock()
	stopInstances--
	if stopInstances > 0 || players > 0 || serverStatus == "offline" {
		mutex.Unlock()
		return
	}
	mutex.Unlock()
	serverStatus = "offline"

	if runtime.GOOS == "linux" {
		err := exec.Command("/bin/bash", "-c", stopMinecraftServerLin).Run()
		if err != nil {
			log.Printf("error stopping minecraft server: %v\n", err)
		}
	} else if runtime.GOOS == "windows" {
		cmdIn.Write([]byte(stopMinecraftServerWin))
		cmdIn.Close()
	} else {
		log.Print("error: OS not supported!")
		os.Exit(1)
	}

	log.Print("*** MINECRAFT SERVER IS SHUTTING DOWN!")
	timeLeftUntilTp = minecraftServerStartupTime
}

func setServerStatusOnline() {
	serverStatus = "online"
	log.Print("*** MINECRAFT SERVER IS UP!")
	mutex.Lock()
	stopInstances++
	mutex.Unlock()
	go timer(timeBeforeStoppingEmptyServer, stopEmptyMinecraftServer)
}

func updateTimeleft() {
	if timeLeftUntilTp > 0 {
		timeLeftUntilTp--
		go timer(1, updateTimeleft)
	}
}

func printDataUsage() {
	mutex.Lock()
	if dataCountBytesToServer != 0 || dataCountBytesToClients != 0 {
		logger(fmt.Sprintf("data/s: %8.3f KB/s to clients | %8.3f KB/s to server\n", dataCountBytesToClients/1024, dataCountBytesToServer/1024))
		dataCountBytesToServer = 0
		dataCountBytesToClients = 0
	}
	mutex.Unlock()
	go timer(1, printDataUsage)
}

func main() {
	fmt.Println(strings.Join(info[1:4], "\n"))
	for {
		dockSocket, err := net.Listen("tcp", listenHost+":"+listenPort)
		checkError(err)
		defer func() {
			logger("Closing connection for: dockSocket")
			dockSocket.Close()
		}()
		log.Println("*** listening for new clients to connect...")
		printDataUsage()
		for {
			clientSocket, err := dockSocket.Accept()
			if err != nil {
				continue
			}
			go handleClientSocket(clientSocket)
		}
	}

}

func handleClientSocket(clientSocket net.Conn) {
	clientAddress := strings.Split(clientSocket.RemoteAddr().String(), ":")[0]
	logger(fmt.Sprintf("*** from %s:%s to %s:%s", clientAddress, listenPort, targetHost, targetPort))
	if serverStatus == "offline" || serverStatus == "starting" {
		buffer := make([]byte, 1024)
		dataLenght, err := clientSocket.Read(buffer)
		if err != nil {
			logger("error during clientSocket.Read() 1")
			return
		}

		if buffer[dataLenght-1] == 1 {
			if serverStatus == "offline" {
				log.Printf("*** player unknown requested server info from %s:%s to %s:%s\n", clientAddress, listenPort, targetHost, targetPort)
			} else if serverStatus == "starting" {
				log.Printf("*** player unknown requested server info from %s:%s to %s:%s during server startup\n", clientAddress, listenPort, targetHost, targetPort)
			}

		} else if buffer[dataLenght-1] == 2 {
			dataLenght, err = clientSocket.Read(buffer)
			if err != nil {
				logger("error during clientSocket.Read() 2")
				return
			}
			playerName := string(buffer[3:dataLenght])

			if serverStatus == "offline" {
				startMinecraftServer()
				log.Printf("*** %s tryed to join from %s:%s to %s:%s\n", playerName, clientAddress, listenPort, targetHost, targetPort)
				clientSocket.Write(buildMessage(fmt.Sprintf("Server start command issued. Please wait... Time left: %d seconds", timeLeftUntilTp)))
			} else if serverStatus == "starting" {
				log.Printf("*** %s tryed to join from %s:%s to %s:%s during server startup\n", playerName, clientAddress, listenPort, targetHost, targetPort)
				clientSocket.Write(buildMessage(fmt.Sprintf("Server is starting. Please wait... Time left: %d seconds", timeLeftUntilTp)))
			}
		}

		logger(fmt.Sprintf("closing connection for: %s", clientAddress))
		clientSocket.Close()
	}

	if serverStatus == "online" {
		serverSocket, err := net.Dial("tcp", targetHost+":"+targetPort)
		if err != nil {
			logger("error during serverSocket.Dial()")
			return
		}

		connectSocketsAsync(clientSocket, serverSocket)
	}
}

func connectSocketsAsync(client net.Conn, server net.Conn) {
	go clientToServer(client, server)
	go serverToClient(server, client)
}

func clientToServer(source, destination net.Conn) {
	players++
	log.Printf("*** A PLAYER JOINED THE SERVER! - %d players online", players)

	forwardSync(source, destination, false)

	players--
	log.Printf("*** A PLAYER LEFT THE SERVER! - %d players online", players)

	mutex.Lock()
	stopInstances++
	mutex.Unlock()

	go timer(timeBeforeStoppingEmptyServer, stopEmptyMinecraftServer)
}

func serverToClient(source, destination net.Conn) {
	forwardSync(source, destination, true)
}

//forwardSync takes a source and a destination net.Conn and forwards them (plus takes a true or false to know what it is forwarding)
func forwardSync(source, destination net.Conn, isServerToClient bool) {
	data := make([]byte, 1024)
	for {
		source.SetReadDeadline(time.Now().Add(timeBeforeStoppingEmptyServer * time.Second))
		destination.SetWriteDeadline(time.Now().Add(timeBeforeStoppingEmptyServer * time.Second))

		dataLen, err := source.Read(data)
		if err != nil {
			if err == io.EOF {
				logger(fmt.Sprintf("closing %s --> %s because of EOF", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0]))
			} else {
				logger(fmt.Sprintf("error in forward(): %v\nsource: %s\ndestination: %s", err, strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0]))
			}
			source.Close()
			destination.Close()
			break
		}
		destination.Write(data[:dataLen])

		if debug {
			mutex.Lock()
			if isServerToClient {
				dataCountBytesToClients = dataCountBytesToClients + float64(dataLen)
			} else {
				dataCountBytesToServer = dataCountBytesToServer + float64(dataLen)
			}
			mutex.Unlock()
		}
	}
}

//---------------------------utils----------------------------//

func checkError(err error) {
	if err != nil {
		log.Printf("Fatal error: %s", err.Error())
		time.Sleep(time.Duration(5) * time.Second)
		os.Exit(1)
	}
}

//timer takes a time interval and execute a function after that time has passed
func timer(timeleft int, f func()) {
	time.Sleep(time.Duration(timeleft) * time.Second)
	f()
}

func buildMessage(message string) []byte {
	message = "{\"text\":\"" + message + "\"}"
	encodedMessage := append([]byte{byte(len(message) + 2), byte(0), byte(len(message))}, []byte(message)...)
	return encodedMessage
}

func logger(message string) {
	if debug {
		log.Println(message)
	}
}
