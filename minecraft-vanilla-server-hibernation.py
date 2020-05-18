#!/usr/bin/env python3
'''
minecraft-vanilla_server_hibernation.py is used to start and stop automatically a vanilla minecraft server
Copyright (C) 2020  gekigek99
v4.2 (Python)
visit my github page: https://github.com/gekigek99
If you like what I do please consider having a cup of coffee with me at: https://www.buymeacoffee.com/gekigek99
'''
import psutil
import socket
import _thread
import os
from threading import Timer, Lock
from time import sleep
#------------------------modify-------------------------------#

START_MINECRAFT_SERVER = 'cd PATH/TO/SERVERFOLDER; screen -dmS minecraftSERVER nice -19 java -jar minecraft_server.jar'    #set command to start minecraft-server service
STOP_MINECRAFT_SERVER = "screen -S minecraftSERVER -X stuff 'stop\\n'"    #set command to stop minecraft-server service

MINECRAFT_SERVER_STARTUPTIME = 20       #time the server needs until it is fully started
TIME_BEFORE_STOPPING_EMPTY_SERVER = 120  #time the server waits for clients to connect then it issues the stop command to server

#-----------------------advanced------------------------------#

LISTEN_HOST = "0.0.0.0"
LISTEN_PORT = 25555         #the port you will connect to on minecraft client

TARGET_HOST = "127.0.0.1"
TARGET_PORT = 25565         #the port specified on server.properties

DEBUG = False               #if true more additional information is printed

#---------------------do not modify---------------------------#

players = 0
datacountbytes = 0
server_status = "offline"
timelefttillup = MINECRAFT_SERVER_STARTUPTIME
lock = Lock()
stopinstances = 0

def stop_empty_minecraft_server():
    global server_status, STOP_MINECRAFT_SERVER, players, timelefttillup, stopinstances, lock
    with lock:
        stopinstances -= 1
        if stopinstances > 0 or players > 0 or server_status == "offline":
            return
    server_status = "offline"
    os.system(STOP_MINECRAFT_SERVER)
    print('MINECRAFT SERVER IS SHUTTING DOWN!')
    timelefttillup = MINECRAFT_SERVER_STARTUPTIME

def start_minecraft_server():
    global server_status, START_MINECRAFT_SERVER, MINECRAFT_SERVER_STARTUPTIME, players, timelefttillup
    if server_status != "offline":
        return
    server_status = "starting"
    os.system(START_MINECRAFT_SERVER)
    print ('MINECRAFT SERVER IS STARTING!')
    players = 0
    def _set_server_status_online():
        global server_status, stopinstances, lock
        server_status = "online"
        print ('MINECRAFT SERVER IS UP!')
        with lock:
            stopinstances += 1
        Timer(TIME_BEFORE_STOPPING_EMPTY_SERVER, stop_empty_minecraft_server, ()).start()
    def _update_timeleft():
        global timelefttillup
        if timelefttillup > 0:
            timelefttillup-=1
            Timer(1,_update_timeleft, ()).start()
    _update_timeleft()
    Timer(MINECRAFT_SERVER_STARTUPTIME, _set_server_status_online, ()).start()

def printdatausage():
    global datacountbytes, lock
    with lock:
        if datacountbytes != 0:
            print('{:.3f}KB/s'.format(datacountbytes/1024/3))
            datacountbytes = 0
    Timer(3, printdatausage, ()).start()

def main():
    global players, START_MINECRAFT_SERVER, STOP_MINECRAFT_SERVER, server_status, timelefttillup
    print('minecraft-vanilla-server-hibernation v4.2 (Python)')
    print('Copyright (C) 2020 gekigek99')
    print('visit my github page for updates: https://github.com/gekigek99')
    dock_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    dock_socket.setblocking(1)
    dock_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)   #to prevent errno 98 address already in use
    dock_socket.bind((LISTEN_HOST, LISTEN_PORT))
    dock_socket.listen(5)
    print('*** listening for new clients to connect...')
    if DEBUG == True:
        printdatausage()
    while True:
        try:
            client_socket, client_address = dock_socket.accept()        #blocking
            if DEBUG == True:
                print ('*** from {}:{} to {}:{}'.format(client_address[0], LISTEN_PORT, TARGET_HOST, TARGET_PORT))
            if server_status == "offline" or server_status == "starting":
                connection_data_recv = client_socket.recv(64)
                if connection_data_recv[-1] == 2:       #\x02 is the last byte of the first message when player is trying to join the server
                    player_data_recv = client_socket.recv(64)   #here it's reading an other packet containing the player name
                    player_name = player_data_recv[3:].decode('utf-8', errors='replace')
                    if server_status == "offline":
                        print(player_name, 'tryed to join from', client_address[0])
                        start_minecraft_server()
                    if server_status == "starting":
                        print(player_name, 'tryed to join from', client_address[0], 'during server startup')
                        sleep(0.01)     #necessary otherwise it could throw an error: 
                                        #Internal Exception: io.netty.handler.codec.Decoder.Exception java.lang.NullPointerException
                        #the padding to 88 chars is important, otherwise someclients will fail to interpret
                        #(byte 0x0a (equal to \n or new line) is used to put the phrase in the center of the screen)
                        client_socket.sendall(("e\0c{\"text\":\"" + ("Server is starting. Please wait. Time left: " + str(timelefttillup) + " seconds").ljust(88,'\x0a')+"\"}").encode())
                else:
                    if connection_data_recv[-1] == 1:   #\x01 is the last byte of the first message when requesting server info
                        if server_status == "offline":
                            print('player unknown requested server info from', client_address[0])
                        if server_status == "starting":
                            print('player unknown requested server info from', client_address[0], 'during server startup')
                client_socket.shutdown(1)   #sends FIN to client
                client_socket.close()
                continue
            if server_status == "online":    
                server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                server_socket.connect((TARGET_HOST, TARGET_PORT))
                connectsocketsasync(client_socket,server_socket)
        except Exception as e:
            print ('Exception in main(): '+str(e))

def connectsocketsasync(client, server):
    _thread.start_new_thread(clienttoserver, (client, server,))
    _thread.start_new_thread(servertoclient, (server, client,))

def clienttoserver(source, destination):
    global players, TIME_BEFORE_STOPPING_EMPTY_SERVER, stop_empty_minecraft_server, stopinstances, lock
    players +=1
    print ('A PLAYER JOINED THE SERVER! - '+str(players)+' players online')
    forwardsync(source,destination)
    players -= 1
    print ('A PLAYER LEFT THE SERVER! - '+str(players)+' players remaining')
    with lock:
        stopinstances += 1
    Timer(TIME_BEFORE_STOPPING_EMPTY_SERVER, stop_empty_minecraft_server, ()).start()

def servertoclient(source, destination):
    forwardsync(source, destination)

#this thread passes data between connections
def forwardsync(source, destination):
    global datacountbytes, lock
    data = ' '
    source.settimeout(60)
    destination.settimeout(60)
    try:
        while True:
            data = source.recv(1024)
            if not data:                #if there is no data stop listening, this means the socket is closed
                break
            destination.sendall(data)
            with lock:
                datacountbytes += len(data) #to calculate the quantity of data per second
    except IOError as e: 
        if e.errno == 32:               #user/server disconnected normally. has to be catched, because there is a race condition
            return                      #when trying to check if destination.recv does return data
        print('IOError in forward(): ' + str(e))
    except Exception as e:
        print('Exception in forward(): ' + str(e))

if __name__ == '__main__':
    main()
