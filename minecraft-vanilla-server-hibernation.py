#!/usr/bin/env python3

'''
minecraft-vanilla_server_hibernation.py
version 2.0
'''
import psutil
import socket
import _thread
import os
from threading import Timer
from time import sleep
#------------------------modify-------------------------------#

START_MINECRAFT_SERVER = 'cd PATH/TO/SERVERFOLDER; screen -dmS minecraftSERVER nice -19 java -jar minecraft_server.jar'    #set command to start minecraft-server service
STOP_MINECRAFT_SERVER = "screen -S minecraftSERVER -X stuff 'stop\\n'"    #set command to stop minecraft-server service

MINECRAFT_SERVER_STARTUPTIME = 20       #time the server needs until it is fully started
TIME_BEFORE_STOPPING_EMPTY_SERVER = 60  #time the server waits for clients to connect then it issues the stop command to server

#-----------------------advanced------------------------------#

LISTEN_HOST = "0.0.0.0"
LISTEN_PORT = 25555         #the port you will connect to on minecraft client

TARGET_HOST = "127.0.0.1"
TARGET_PORT = 25565         #the port specified on server.properties

DEBUG = False # if true more additional information is printed

#---------------------do not modify---------------------------#

players = 0    
datacountbytes = 0
server_status = "offline"
timelefttillup = MINECRAFT_SERVER_STARTUPTIME

def stop_empty_minecraft_server():
    global server_status, STOP_MINECRAFT_SERVER, players, timelefttillup
    if players > 0 or server_status == "offline":
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
        global server_status
        server_status = "online"
        print ('MINECRAFT SERVER IS UP!')
        Timer(TIME_BEFORE_STOPPING_EMPTY_SERVER, stop_empty_minecraft_server, ()).start()
    def _update_timeleft():
        global timelefttillup
        if timelefttillup > 0: 
            timelefttillup-=1
            Timer(1,_update_timeleft, ()).start()
    _update_timeleft()
    Timer(MINECRAFT_SERVER_STARTUPTIME, _set_server_status_online, ()).start()

def printdatausage():
    global datacountbytes
    if datacountbytes != 0:
        print('{:.3f}KB/s'.format(datacountbytes/1024/3))
        datacountbytes = 0
    Timer(3, printdatausage, ()).start()

def main():
    global players, START_MINECRAFT_SERVER, STOP_MINECRAFT_SERVER, server_status, timelefttillup
    dock_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    dock_socket.setblocking(1)
    dock_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1) #to prevent errno 98 address already in use
    dock_socket.bind((LISTEN_HOST, LISTEN_PORT))
    dock_socket.listen(5)
    if DEBUG == True:
        printdatausage()
    while True:
        try:
            client_socket, client_address = dock_socket.accept() # blocking
            if DEBUG == True:
                print ('*** from {}:{} to {}:{}'.format(client_address[0], LISTEN_PORT, TARGET_HOST, TARGET_PORT))
            if server_status == "offline":
                connection_data_recv = client_socket.recv(64)
                player_data_recv = client_socket.recv(64)
                player_name = player_data_recv[3:].decode('utf-8', errors='replace')
                if player_name == '':
                    player_name = 'player unknown'
                if connection_data_recv[-1] == 2:     #\x02 is the last byte when player is trying to join the server
                    print(player_name, 'wants to join from', client_address[0])
                    start_minecraft_server()
                elif connection_data_recv[-1] == 1:   #\x01 is the last byte when requesting server info
                    print(player_name, 'requested server info from', client_address[0])
                    client_socket.shutdown(1)
                    client_socket.close()
                    continue
                else:
                    client_socket.shutdown(1)
                    client_socket.close()
                    continue
            if server_status == "starting":
                sleep(0.01) #necessary otherwise it throws an error: Internal Exception: io.netty.handler.codec.Decoder.Exception java.lang.NullPointerException
                print(player_name, 'connected from', client_address[0], 'while starting')
                #the padding to 88 chars is important, otherwise someclients will fail to interpret (byte 0x0a (equal to \n or new line) is used to put the phrase in the center of the screen)
                client_socket.sendall(("e\0c{\"text\":\"" + ("Server is starting. Please wait. Time left: " + str(timelefttillup) + " seconds").ljust(88,'\x0a')+"\"}").encode())
                client_socket.shutdown(1) # sends FIN to client
                client_socket.close()
                continue
            server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            server_socket.connect((TARGET_HOST, TARGET_PORT))
            connectsocketsasync(client_socket,server_socket)
        except Exception as e:
            print ('Exception in main(): '+str(e))

def connectsocketsasync(client, server):
    _thread.start_new_thread(clienttoserver, (client, server,))
    _thread.start_new_thread(servertoclient, (server, client,))

def clienttoserver(source, destination):
    global players, TIME_BEFORE_STOPPING_EMPTY_SERVER, stop_empty_minecraft_server
    players +=1
    print ('A PLAYER JOINED THE SERVER! - '+str(players)+' players online')
    forwardsync(source,destination)
    players -= 1
    print ('A PLAYER LEFT THE SERVER! - '+str(players)+' players remaining')
    Timer(TIME_BEFORE_STOPPING_EMPTY_SERVER, stop_empty_minecraft_server, ()).start()

def servertoclient(source, destination):
    forwardsync(source, destination)

#this thread passes data between connections
def forwardsync(source, destination):
    global datacountbytes
    data = ' '
    source.settimeout(60)
    destination.settimeout(60)
    try:
        while True:
            data = source.recv(1024)
            if not data: #if there is no data stop listening, this means the socket is closed
                break
            destination.sendall(data)
            datacountbytes += len(data) #to calculate the quantity of data per second
    except IOError as e: 
        if e.errno == 32: # user/server disconnected normally. has to be catched, because there is a race condition when trying to check if destination.recv does return data
            return
        print('IOError in forward(): ' + str(e))
    except Exception as e:
        print('Exception in forward(): ' + str(e))

if __name__ == '__main__':
    main()
