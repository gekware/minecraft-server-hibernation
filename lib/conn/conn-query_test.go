package conn

import (
	"fmt"
	"testing"

	"github.com/dreamscached/minequery/v2"

	"msh/lib/config"
)

func Test_QueryFull(t *testing.T) {
	config.ListenHost, config.ListenPort = "127.0.0.1", 24444

	go HandlerQuery()

	minequery.WithUseStrict(true)

	for i := 0; i < 2; i++ {
		fmt.Println("--------------------")

		res, err := minequery.QueryFull(config.ListenHost, config.ListenPort)
		if err != nil {
			t.Fatalf(err.Error())
		}

		fmt.Printf("result: %+v\n", res)
	}
}

func Test_QueryBasic(t *testing.T) {
	config.ListenHost, config.ListenPort = "127.0.0.1", 24444

	go HandlerQuery()

	minequery.WithUseStrict(true)

	for i := 0; i < 2; i++ {
		fmt.Println("--------------------")

		res, err := minequery.QueryBasic(config.ListenHost, config.ListenPort)
		if err != nil {
			t.Fatalf(err.Error())
		}

		fmt.Printf("result: %+v\n", res)
	}
}
