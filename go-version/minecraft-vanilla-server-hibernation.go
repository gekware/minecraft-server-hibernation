/*
minecraft-vanilla_server_hibernation is used to start and stop automatically a vanilla minecraft server
Copyright (C) 2020  gekigek99

v1.1 (Go)
visit my github page: https://github.com/gekigek99
Script slightly modified for Docker usage by github.com/lubocode
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

//------------------------modify-------------------------------//

var startminecraftserver string // To modify this, have a look at the default values (third argument) of the flags in main() or pass the corresponding command line arguments.

const stopminecraftserver = "screen -S minecraftSERVER -X stuff 'stop\\n'"

const minecraftserverstartuptime = 20
const timebeforestoppingemptyserver = 120

//-----------------------advanced------------------------------//

const listenhost = "0.0.0.0"
const listenport = "25555"

const targethost = "127.0.0.1"
const targetport = "25565"

const debug = false

//---------------------do not modify---------------------------//

var players int = 0
var datacountbytestoserver, datacountbytestoclients float64 = 0, 0
var serverstatus string = "offline"
var timelefttillup int = minecraftserverstartuptime
var stopinstances int = 0
var mutex = &sync.Mutex{}

//StopEmptyMinecraftServer stops the minecraft server
func StopEmptyMinecraftServer() {
	mutex.Lock()
	stopinstances--
	if stopinstances > 0 || players > 0 || serverstatus == "offline" {
		return
	}
	mutex.Unlock()
	serverstatus = "offline"
	err := exec.Command("/bin/bash", "-c", stopminecraftserver).Run()
	if err != nil {
		log.Printf("error stopping minecraft server: %v\n", err)
	}
	log.Print("*** MINECRAFT SERVER IS SHUTTING DOWN!")
	timelefttillup = minecraftserverstartuptime
}

//StartMinecraftServer starts the minecraft server
func StartMinecraftServer() {
	if serverstatus != "offline" {
		return
	}
	serverstatus = "starting"
	err := exec.Command("/bin/bash", "-c", startminecraftserver).Run()
	if err != nil {
		log.Printf("error starting minecraft server: %v\n", err)
	}
	log.Print("*** MINECRAFT SERVER IS STARTING!")
	players = 0
	UpdateTimeleft()
	go Timer(minecraftserverstartuptime, SetServerStatusOnline)
}

//SetServerStatusOnline sets the server status online
func SetServerStatusOnline() {
	serverstatus = "online"
	log.Print("*** MINECRAFT SERVER IS UP!")
	mutex.Lock()
	stopinstances++
	mutex.Unlock()
	go Timer(timebeforestoppingemptyserver, StopEmptyMinecraftServer)
}

//UpdateTimeleft updates the global variable timelefttillup
func UpdateTimeleft() {
	if timelefttillup > 0 {
		go Timer(1, UpdateTimeleft)
		timelefttillup--
	}
}

func printdatausage() {
	if debug == true {
		mutex.Lock()
		if datacountbytestoserver != 0 || datacountbytestoclients != 0 {
			log.Printf("data/s: %8.3f KB/s to clients | %8.3f KB/s to server\n", datacountbytestoclients/1024, datacountbytestoserver/1024)
			datacountbytestoserver = 0
			datacountbytestoclients = 0
		}
		mutex.Unlock()
		go Timer(1, printdatausage)
	}
}

func main() {
	var minRAM string
	var maxRAM string
	var mcPath string
	var mcFile string

	flag.StringVar(&minRAM, "minRAM", "512M", "Specify minimum amount of RAM.")
	flag.StringVar(&maxRAM, "maxRAM", "2G", "Specify maximum amount of RAM.")
	flag.StringVar(&mcPath, "mcPath", "/minecraftserver/", "Specify path of Minecraft folder.")
	flag.StringVar(&mcFile, "mcFile", "minecraft_server.jar", "Specify name of Minecraft .jar file")
	flag.Parse()
	minRAM = "-Xms" + minRAM
	maxRAM = "-Xmx" + maxRAM

	startminecraftserver = "cd " + mcPath + "; screen -dmS minecraftSERVER nice -19 java " + minRAM + " " + maxRAM + " -jar " + mcFile

	fmt.Println("minecraft-vanilla-server-hibernation v1.1 (Go)")
	fmt.Println("Copyright (C) 2020 gekigek99")
	fmt.Println("Original creators github page: https://github.com/gekigek99")
	fmt.Println("Modified for docker usage by: https://github.com/lubocode")
	fmt.Println("Container started with the following arguments: \n\tminRAM:" + minRAM + " maxRAM:" + maxRAM + " mcPath:" + mcPath + " mcFile:" + mcFile)

	for {
		docksocket, err := net.Listen("tcp", listenhost+":"+listenport)
		CheckError(err)
		defer StopEmptyMinecraftServer()
		defer func() {
			if debug == true {
				log.Println("Closing connection for: docksocket")
			}
			docksocket.Close()
		}()
		log.Println("*** listening for new clients to connect...")
		printdatausage()
		for {
			clientsocket, err := docksocket.Accept()
			if err != nil {
				continue
			}
			go handleclientsocket(clientsocket)
		}
	}

}

func handleclientsocket(clientsocket net.Conn) {
	clientsocketremoteaddr := strings.Split(clientsocket.RemoteAddr().String(), ":")[0]
	if serverstatus == "offline" || serverstatus == "starting" {
		buffer := make([]byte, 1024)
		datalenght, err := clientsocket.Read(buffer)
		if err != nil {
			if debug == true {
				log.Printf("error during clientsocket.Read() 1")
			}
			return
		}
		locationpattern, foundpattern := FindPattern(buffer[:datalenght], []byte{2, 11, 0, 9}) //for me is 2,11,0,9 don't know if it's a general rule
		if buffer[datalenght-1] == 2 || foundpattern {
			playername := ""
			if foundpattern == false {
				datalenght, err = clientsocket.Read(buffer)
				if err != nil {
					if debug == true {
						log.Printf("error during clientsocket.Read() 2")
					}
					return
				}
				playername = string(buffer[3:datalenght])
			} else {
				playername = string(buffer[locationpattern+4 : datalenght])
			}
			if serverstatus == "offline" {
				log.Printf("*** %s tried to join from %s:%s to %s:%s\n", playername, clientsocketremoteaddr, listenport, targethost, targetport)
				StartMinecraftServer()
			}
			if serverstatus == "starting" {
				log.Printf("*** %s tried to join from %s:%s to %s:%s during server startup\n", playername, clientsocketremoteaddr, listenport, targethost, targetport)
				message := BuildMessage(fmt.Sprintf("Server is starting. Please wait. Time left: %d seconds", timelefttillup))
				clientsocket.Write(message)
			}
		} else {
			locationpattern, foundpattern := FindPattern(buffer[:datalenght], []byte{1, 1, 0})
			if buffer[datalenght-1] == 1 || (locationpattern == datalenght-3 && foundpattern) { //sometimes the requests ends with  [...1 1 0] and not with [...1]
				if serverstatus == "offline" {
					log.Printf("*** player unknown requested server info from %s:%s to %s:%s\n", clientsocketremoteaddr, listenport, targethost, targetport)
				}
				if serverstatus == "starting" {
					log.Printf("*** player unknown requested server info from %s:%s to %s:%s during server startup\n", clientsocketremoteaddr, listenport, targethost, targetport)
				}
			}
		}
		if debug == true {
			log.Println("closing connection for: ", clientsocketremoteaddr)
		}
		clientsocket.Close()
	}
	if serverstatus == "online" {
		serversocket, err := net.Dial("tcp", targethost+":"+targetport)
		if err != nil {
			if debug == true {
				log.Printf("error during serversocket.Dial()")
			}
			return
		}
		connectsocketsasync(clientsocket, serversocket)
	}
}

func connectsocketsasync(client net.Conn, server net.Conn) {
	go ClientToServer(client, server)
	go ServerToClient(server, client)
}

//ClientToServer manages the client to server connection
func ClientToServer(source, destination net.Conn) {
	players++
	log.Printf("*** A PLAYER JOINED THE SERVER! - %d players online", players)
	ForwardSync(source, destination, false)
	players--
	log.Printf("*** A PLAYER LEFT THE SERVER! - %d players online", players)
	mutex.Lock()
	stopinstances++
	mutex.Unlock()
	go Timer(timebeforestoppingemptyserver, StopEmptyMinecraftServer)
}

//ServerToClient manages the server to client connection
func ServerToClient(source, destination net.Conn) {
	ForwardSync(source, destination, true)
}

//ForwardSync takes a source and a destination net.Conn and forwards them (plus takes a true or false to know what it is forwarding)
func ForwardSync(source, destination net.Conn, servertoclient bool) {
	data := make([]byte, 1024)
	for {
		source.SetReadDeadline(time.Now().Add(20 * time.Second))
		destination.SetWriteDeadline(time.Now().Add(20 * time.Second))
		lenghtdata, err := source.Read(data)
		if err != nil {
			if debug == true {
				if err == io.EOF {
					log.Printf("closing %s --> %s because of EOF", strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0])
				} else {
					log.Printf("error in forward(): %v\nsource: %s\ndestination: %s", err, strings.Split(source.RemoteAddr().String(), ":")[0], strings.Split(destination.RemoteAddr().String(), ":")[0])
				}
			}
			source.Close()
			destination.Close()
			break
		}
		destination.Write(data[:lenghtdata])
		if debug == true {
			mutex.Lock()
			if servertoclient == true {
				datacountbytestoclients = datacountbytestoclients + float64(lenghtdata)
			} else {
				datacountbytestoserver = datacountbytestoserver + float64(lenghtdata)
			}
			mutex.Unlock()
		}
	}
}

//----------------------func tools-----------------------------//

//CheckError takes an error variable and exit if not nil
func CheckError(err error) {
	if err != nil {
		log.Printf("Fatal error: %s", err.Error())
		time.Sleep(time.Duration(5) * time.Second)
		os.Exit(1)
	}
}

//Timer takes a time interval and execute a function after that time has passed
func Timer(timeleft int, f func()) {
	time.Sleep(time.Duration(timeleft) * time.Second)
	f()
}

//BuildMessage takes a string and returns a []byte
func BuildMessage(message string) []byte {
	message = PaddString(message, "\x0a", 88)
	message = "e\x00c{\"text\":\"" + message + "\"}"
	messagebyte := []byte(message)
	return messagebyte
}

//PaddString takes a string a padding and a lenght and returns a string
func PaddString(text, padding string, lenght int) string {
	for i := len(text); i < lenght; i++ {
		text = text + padding
	}
	return text
}

//FindPattern returns true and the location of the pattern found, or returns false and -1
func FindPattern(array, pattern []byte) (int, bool) {
	lenarray := len(array)
	lenpattern := len(pattern)
	for i := 0; i <= lenarray-lenpattern; i++ {
		for j := 0; j < lenpattern; j++ {
			if array[i+j] != pattern[j] {
				break
			}
			if j == lenpattern-1 {
				return i, true
			}
		}
	}
	return -1, false
}
