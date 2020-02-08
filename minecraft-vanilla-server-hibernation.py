#!/usr/bin/env python3

'''
minecraft-vanilla_server_hibernation.py
version 2.0
'''

import socket
import sys
import _thread
import os
import time

#------------------------modify-------------------------------#

START_MINECRAFT_SERVER = 'screen -dmS minecraftSERVER nice -19 java -jar minecraft_server.jar'    #set command to start minecraft-server service
STOP_MINECRAFT_SERVER = "screen -S minecraftSERVER -X stuff 'stop\\n'"    #set command to stop minecraft-server service

LISTEN_HOST = "0.0.0.0"
LISTEN_PORT = 25565         #the port you will connect to on minecraft client

TARGET_HOST = "127.0.0.1"
TARGET_PORT = 25555         #the port specified on server.properties

MINECRAFT_SERVER_STARTUPTIME = 120 # time the server needs until it is fully started

TIME_BEFORE_STOPPING_EMPTY_SERVER = 20 

#-----------------------advanced------------------------------#

WRITE_LOG = False               #for debug

TIMEOUT_SOCKET = 240             #after which it is raised socket.timeout exception (to prevent the case in which someone clicks the first time
                                #to start up to server and then doesn't click a second time to enter the world)

#---------------------do not modify---------------------------#

restart_flag = False            #used when there is a disconnection of a player, if true it signals the need to restart this program

players = 0                     #declared here because it's a global variable

firstlaunch = True              #used to know that the minecraft-server service is not running and this program needs to start it

datacountbytes = 0
nowtime = 0
nexttime = 0

if WRITE_LOG == True:
    f = open('log.txt', 'w')


def main():
    _thread.start_new_thread(server, () )
    lock = _thread.allocate_lock()
    lock.acquire()              #I don't know why it is needed to be called twice lock.acquire, but it works... ;)
    lock.acquire()


def server(*settings):
    global players, firstlaunch, restart_flag, TIMEOUT_SOCKET, START_MINECRAFT_SERVER, STOP_MINECRAFT_SERVER

    try:
        dock_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        dock_socket.settimeout(TIMEOUT_SOCKET)
        dock_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)           #to prevent errno 98 address already in use
        print('waiting to bind socket...')
        dock_socket.bind((LISTEN_HOST, LISTEN_PORT))                                #listen
        print('socket binded')
        dock_socket.listen(5)
        while True:
            print ('*** listening on {}:{}'.format(LISTEN_HOST, LISTEN_PORT))
            client_socket, client_address = dock_socket.accept()                    #accept
            print ('*** from {}:{} to {}:{}'.format(client_address[0], LISTEN_PORT, TARGET_HOST, TARGET_PORT))
            server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            server_socket.connect((TARGET_HOST, TARGET_PORT))                       #connect
            _thread.start_new_thread(forward, (client_socket, server_socket, "client-->server",))   #a new thread is started to send data from client to server
            _thread.start_new_thread(forward, (server_socket, client_socket, "server-->client",))   #a new thread is started to send data from server to client
            print ('A PLAYER JOINED THE SERVER!')           #if the socket was accepted without errors, it means that a player has just connected
            if WRITE_LOG == True:
                f.write('A PLAYER JOINED THE SERVER!\n')
            players += 1
            print (str(players) + ' players')

    except socket.timeout:      #after the set socket.settimeout the program checks if there are no players, if so it send a stop minecraft-server service to make sure the server is down
        print('connection timeout')
        time.sleep(TIME_BEFORE_STOPPING_EMPTY_SERVER)
        if players <= 0:
            print('no players on server and connection timed out...')
            print('ISSUING STOP COMMAND TO MINECRAFT-SERVER!')
            os.system(STOP_MINECRAFT_SERVER)
            print('STOP COMMAND TO MINECRAFT-SERVER ISSUED ---> MINECRAFT SERVER IS DOWN!')
            firstlaunch = True
            if restart_flag == True:
                print('restarting ' + str(os.path.basename(__file__)) + '...')      #after all player exit the game restart_flag is set to true because there is the possibility 
                os.execl(str(os.path.basename(__file__)), 'python3')                #that the player has lost connection and that 1 thread is not closed successfully
                                                                                    #(remember: for each player there are 2 threads, 1 server-->client and 1 client-->server)
                                                                                    #the restart commands is used to avoid build-up of non-useful threads in the case
                                                                                    #of a player losing connection multiple times
    except IOError as e:
        if e.errno == 111 and firstlaunch == True:                                  #errno:111 is returned when there is the first connection (and minecraft-server is down)
            print('errno_111 && firstlaunch == True ---> FIRST SERVER CONNECTION')
            print ('starting minecraft-server service')
            os.system(START_MINECRAFT_SERVER)
            print('loading world, wait '+str(MINECRAFT_SERVER_STARTUPTIME)+' seconds...')
            players = 0
            time.sleep(MINECRAFT_SERVER_STARTUPTIME)      #needed for launching minecraft-server, it can vary on how fast is your server, and probably 2 or 3 seconds is enough (but higher number prevents a player from entering when the server is still loading the world)
            print ('MINECRAFT SERVER IS UP!')
            firstlaunch = False
        elif e.errno == 111 and firstlaunch == False:       #if the stop minecraft-server command has just been issued and if this script is not fast enough to bind again the socket there could be some errors, this IF prevents them
            print('errno_111 ---> not ready yet')
        else:
            print('IOError.errno: ' + str(e.errno))         #the else (always put it at the end!!!) caches all other possible errors to make the debug easyer

    except Exception as e:
        print('exception in function server(): ' + str(e))  #this is to catch the most general errors (not only IOErrors)

    finally:
        _thread.start_new_thread(server, () )               #after the cycle has been completed start a new server thread to listen for new players to connect


def forward(source, destination, description):              #this thread is the one that manages alle data passing through this program
    global players, firstlaunch, datacountbytes, nowtime, nexttime, f, restart_flag, LISTEN_HOST, LISTEN_PORT, WRITE_LOG

    try:
        data = ' '              #so that it is possible to enter the while loop
        while data:
            data = source.recv(1024)
            nowtime = time.time()
            #print('{}: {}'.format(description, data.decode('ascii', 'replace')))   #commented out because it's non useful and just resource consuming on terminal
            if WRITE_LOG == True:
                f.write(str(nowtime) + '>' + str(description) + data.decode('ascii', 'replace') + '\n' )
            if data:
                destination.sendall(data)
                datacountbytes += len(data)                 #to calculate the quantity of data per second
                if nowtime >= nexttime:
                    if WRITE_LOG == True:
                        print('{:.3f}KB/s'.format(datacountbytes/1024))
                    datacountbytes = 0
                    nexttime = nowtime + 1
            else:                                           #if there is no data close the socket and declare that a player has left the server
                source.shutdown(socket.SHUT_RD)
                destination.shutdown(socket.SHUT_WR)

                players -= 1
                print('A PLAYER LEFT THE SERVER!')
                restart_flag = True                         #(as said before) this flag is set in order to prevent problems derived from a player losing connection
                print('restart_flag set')
                if WRITE_LOG == True:
                    f.write('A PLAYER LEFT THE SERVER\n') 
                print (str(players) + ' players')
                time.sleep(TIME_BEFORE_STOPPING_EMPTY_SERVER)
                if players <= 0:
                    print('ISSUING STOP COMMAND TO MINECRAFT-SERVER!')
                    os.system(STOP_MINECRAFT_SERVER)        #stops minecraft-server service if there ar no players
                    print('STOP COMMAND TO MINECRAFT-SERVER ISSUED ---> MINECRAFT SERVER IS DOWN!')
                    firstlaunch = True
                    print ('*** listening on {}:{}'.format(LISTEN_HOST, LISTEN_PORT))

    except IOError as e:                                    #if error number 107 is caught it means that a player has left the game and restart_flag is set in order to prevent problems derived from a player losing connection
        if e.errno == 107:
            print('errno_107 ---> a player just disconnected')
            restart_flag = True
            print('restart_flag set')
        else:
            print('IOError: ' + str(e))

    except Exception as e:
        print('Exception in function forward(): ' + str(e))


if __name__ == '__main__':
    main()
