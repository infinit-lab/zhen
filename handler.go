package main

import (
	"github.com/infinit-lab/yolanda/logutils"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func handle(r *request, conn net.Conn, timer *time.Timer) (isClose bool) {
	isClose = true
	switch r.method {
	case http.MethodConnect:
		connect(r, conn, timer)
	default:
		transfer(r, conn)
	}
	connections, ok := r.headers["Connection"]
	if ok && len(connections) > 0{
		if strings.ToLower(connections[0]) == "keep-alive" {
			isClose = false
		}
	}
	return
}

func connect(r *request, conn net.Conn, timer *time.Timer) {
	addr, err := net.ResolveTCPAddr("tcp", r.url)
	if err != nil {
		logutils.Error("Failed to ResolveTCPAddr. error: ", err)
		writeError(conn, err.Error(), http.StatusBadGateway)
		return
	}
	tcpConn, err := net.DialTCP("tcp4", nil, addr)
	if err != nil {
		logutils.Error("Failed to DialTCP. error: ", err)
		writeError(conn, err.Error(), http.StatusBadGateway)
		return
	}
	go func() {
		for {
			buffer := make([]byte, 1448)
			recvLen, err := tcpConn.Read(buffer)
			if err != nil {
				logutils.Info("Failed to Read. error: ", err)
				_ = conn.Close()
				break
			}
			_, _  = conn.Write(buffer[:recvLen])
		}
	}()

	var rsp response
	rsp.status = "Connection Established"
	rsp.statusCode = http.StatusOK
	rsp.version = r.version
	rsp.headers = make(map[string][]string)
	write(conn, &rsp)

	for {
		buffer := make([]byte, 1448)
		recvLen, err := conn.Read(buffer)
		if err != nil {
			logutils.Info("Failed to Read. error: ", err)
			_ = tcpConn.Close()
			break
		}
		timer.Reset(time.Second * 180)
		_, _ = tcpConn.Write(buffer[:recvLen])
	}
}

func transfer(r *request, conn net.Conn) {
	client := new(http.Client)
	req, err := http.NewRequest(r.method, r.url, strings.NewReader(string(r.body)))
	if err != nil {
		logutils.Error("Failed to NewRequest. error: ", err)
		writeError(conn, err.Error(), http.StatusBadGateway)
		return
	}
	req.Header = r.headers
	res, err := client.Do(req)
	if err != nil {
		logutils.Error("Failed to Do. error: ", err)
		writeError(conn, err.Error(), http.StatusBadGateway)
		return
	}
	var rsp response
	rsp.status = res.Status
	rsp.statusCode = res.StatusCode
	rsp.version = r.version
	rsp.headers = res.Header
	rsp.body, err = ioutil.ReadAll(res.Body)
	defer func() {
		_ = res.Body.Close()
	}()
	if err != nil {
		logutils.Error("Failed to ReadAll. error: ", err)
		writeError(conn, err.Error(), http.StatusBadGateway)
		return
	}
	write(conn, &rsp)
}

type response struct {
	status string
	statusCode int
	version string
	headers map[string][]string
	body []byte
}

func write(conn net.Conn, r *response) {
	var buffer string
	buffer = r.version + " " + strconv.Itoa(r.statusCode) + " " + r.status + "\r\n"
	for key, values := range r.headers {
		if strings.ToLower(key) == "content-length" {
			continue
		}
		for _, value := range values {
			buffer += key + ":" + value + "\r\n"
		}
	}
	if len(r.body) > 0 {
		buffer += "Content-Length:" + strconv.Itoa(len(r.body)) + "\r\n"
	}
	buffer += "\r\n"
	buffer += string(r.body)
	_, _ = conn.Write([]byte(buffer))
}

func writeError(conn net.Conn, err string, statusCode int) {
	var r response
	r.status = err
	r.statusCode = statusCode
	r.version = "HTTP/1.1"
	r.headers = make(map[string][]string)
	r.headers["Server"] = []string{"zhen"}
	write(conn, &r)
}
