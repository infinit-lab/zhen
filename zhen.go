package main

import (
	"fmt"
	"github.com/infinit-lab/yolanda/config"
	"github.com/infinit-lab/yolanda/logutils"
	"net"
	"time"
)

type proxy struct {
	listener net.Listener
}

var p proxy

func main() {
	port := config.GetInt("server.port")
	if port == 0 {
		port = 7071
	}

	var err error
	p.listener, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		logutils.Error("Failed to Listen. error: ", err)
		return
	}
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			logutils.Error("Failed to Accept. error: ", err)
			return
		}
		go func() {
			var cache string
			timer := time.NewTimer(time.Second * 180)
			quitChan := make(chan int)
			go func() {
				select {
				case <- timer.C:
					_ = conn.Close()
				case <- quitChan:
					return
				}
			}()
			for {
				buffer := make([]byte, 1448)
				recvLen, err := conn.Read(buffer)
				if err != nil {
					logutils.Info("Failed to Read. error: ", err)
					break
				}
				timer.Reset(time.Second * 180)
				cache += string(buffer[:recvLen])
				req, err := parse(cache)
				if err != nil {
					logutils.Trace("Failed to parse. error: ", err)
					continue
				}
				if handle(req, conn, timer) {
					_ = conn.Close()
				}
				cache = ""
			}
			timer.Stop()
			close(quitChan)
		}()
	}
}

