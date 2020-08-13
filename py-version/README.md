### WINDOWS
##### (soon an appropriate version will be released):
windows does not support the command "screen" therefore you will need to
#### add:
```Python
from subprocess import Popen, PIPE, STDOUT
```
#### replace:
```Python
os.system(START_MINECRAFT_SERVER)
#with
start_minecraft_server.p = Popen(['java', '-Xmx1024M', '-Xms1024M', '-jar', 'server.jar', 'nogui'], stdout=PIPE, stdin=PIPE, stderr=STDOUT)
```
```Python
os.system(STOP_MINECRAFT_SERVER)
#with
start_minecraft_server.p.communicate(input=b'stop')[0]
```
#### remove:
```Python
START_MINECRAFT_SERVER	#(parameter)
STOP_MINECRAFT_SERVER	#(parameter)
```
