package conn

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"msh/lib/config"
	"msh/lib/errco"
	"msh/lib/servstats"
)

type startingMessageConn struct{ net.Conn }

func (startingMessageConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 25555}
}

func Test_HandlerClientConn_startingMessage(t *testing.T) {
	msh := &config.ConfigRuntime.Msh
	savedMessage, savedSuspend := msh.MsgStarting, msh.SuspendAllow
	savedWhitelist, savedImport := msh.Whitelist, msh.WhitelistImport
	savedTimeout := msh.TimeBeforeStoppingEmptyServer
	stats := servstats.Stats
	savedStatus, savedError, savedProgress := stats.Status, stats.MajorError, stats.LoadProgress
	savedWarmUpTime, savedTimer := stats.WarmUpTime, stats.FreezeTimer
	t.Cleanup(func() {
		stats.FreezeTimer.Stop()
		msh.MsgStarting, msh.SuspendAllow = savedMessage, savedSuspend
		msh.Whitelist, msh.WhitelistImport = savedWhitelist, savedImport
		msh.TimeBeforeStoppingEmptyServer = savedTimeout
		stats.Status, stats.MajorError, stats.LoadProgress = savedStatus, savedError, savedProgress
		stats.WarmUpTime, stats.FreezeTimer = savedWarmUpTime, savedTimer
	})

	// An already-starting server needs no process launch or resume in WarmMS.
	msh.SuspendAllow, msh.WhitelistImport, msh.Whitelist = false, false, nil
	msh.TimeBeforeStoppingEmptyServer = 3600
	stats.Status, stats.MajorError = errco.SERVER_STATUS_STARTING, nil
	stats.FreezeTimer = time.NewTimer(time.Hour)
	stats.FreezeTimer.Stop()

	tests := []struct {
		name, message, progress, want string
	}{
		{"default", "Server start command issued. Please wait...", "0%", "Server start command issued. Please wait... [loading 0%]"},
		{"custom text", "§6Attendi \"Mondo\"…\nCaricamento", "42%", "§6Attendi \"Mondo\"…\nCaricamento [loading 42%]"},
		{"empty message", "", "87%", " [loading 87%]"},
		{"empty progress", "Almost ready.", "", "Almost ready. [loading ]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			msh.MsgStarting, stats.LoadProgress = test.message, test.progress
			server, client := net.Pipe()
			done := make(chan struct{})
			t.Cleanup(func() {
				client.Close()
				server.Close()
				<-done
			})
			go func() {
				defer close(done)
				HandlerClientConn(startingMessageConn{server})
			}()
			if err := client.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
				t.Fatal(err)
			}
			request := append(mountHandshake(758, "localhost", 25555, 2), 6, 0, 4, 'T', 'e', 's', 't')
			if _, err := client.Write(request); err != nil {
				t.Fatal(err)
			}
			response, err := io.ReadAll(client)
			if err != nil {
				t.Fatal(err)
			}
			<-done
			want := buildMessage(errco.CLIENT_REQ_JOIN, test.want)
			if !bytes.Equal(response, want) {
				t.Errorf("starting response = %q, want %q", response, want)
			}
		})
	}
}
