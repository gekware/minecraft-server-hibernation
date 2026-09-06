package conn

import (
	"fmt"
	"testing"
	"time"

	"github.com/dreamscached/minequery/v2"

	"msh/lib/config"
)

func Test_QueryFull(t *testing.T) {
	config.MshHost, config.MshPortQuery = "127.0.0.1", 25555

	go HandlerQuery()

	// give HandlerQuery the time to open the udp listener before querying it
	// (net.ListenPacket is called inside the goroutine and there is no readiness signal)
	time.Sleep(200 * time.Millisecond)

	minequery.WithUseStrict(true)

	for i := 0; i < 3; i++ {
		fmt.Println("--------------------")

		res, err := minequery.QueryFull(config.MshHost, config.MshPortQuery)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("result: %+v\n", res)

		time.Sleep(time.Second)
	}
}

func Test_QueryBasic(t *testing.T) {
	config.MshHost, config.MshPortQuery = "127.0.0.1", 25555

	go HandlerQuery()

	// give HandlerQuery the time to open the udp listener before querying it
	// (net.ListenPacket is called inside the goroutine and there is no readiness signal)
	time.Sleep(200 * time.Millisecond)

	minequery.WithUseStrict(true)

	for i := 0; i < 3; i++ {
		fmt.Println("--------------------")

		res, err := minequery.QueryBasic(config.MshHost, config.MshPortQuery)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("result: %+v\n", res)

		time.Sleep(time.Second)
	}
}
