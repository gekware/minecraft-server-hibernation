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
	res, err := minequery.QueryFull(config.ListenHost, config.ListenPort)
	if err != nil {
		t.Fatalf(err.Error())
	}

	fmt.Printf("%+v\n", res)
}
