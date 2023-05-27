module msh

go 1.19

require (
	github.com/chzyer/readline v1.5.1
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dreamscached/minequery/v2 v2.4.1
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/shirou/gopsutil v3.21.11+incompatible
	golang.org/x/image v0.7.0
	golang.org/x/sys v0.7.0
)

require (
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/stretchr/testify v1.8.3 // indirect
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/tklauser/numcpus v0.6.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.3 // indirect
	golang.org/x/text v0.9.0 // indirect
)

replace github.com/chzyer/readline => ./gitmod/readline
