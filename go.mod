module msh

go 1.24.0

require (
	github.com/chzyer/readline v1.5.1
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dreamscached/minequery/v2 v2.5.0
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/shirou/gopsutil v3.21.11+incompatible
	golang.org/x/image v0.21.0
	golang.org/x/sys v0.41.0
)

require (
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/text v0.34.0 // indirect
)

replace github.com/chzyer/readline => ./gitmod/readline
