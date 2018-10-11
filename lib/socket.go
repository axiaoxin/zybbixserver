package lib

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"time"
)

func connectTCPServer(addr string) (conn *net.TCPConn, err error) {

	type DialResp struct {
		Conn  *net.TCPConn
		Error error
	}

	// Open connection to remote host
	iaddr, err := net.ResolveTCPAddr("tcp", addr)

	// dial tcp and handle timeouts
	ch := make(chan DialResp)

	go func() {
		conn, err = net.DialTCP("tcp", nil, iaddr)
		ch <- DialResp{Conn: conn, Error: err}
	}()

	select {
	case <-time.After(5 * time.Second):
		err = fmt.Errorf("Connection Timeout")

	case resp := <-ch:
		if resp.Error != nil {
			err = resp.Error
			break
		}

		conn = resp.Conn
	}

	return
}

func readFromConn(conn *net.TCPConn) (res []byte, err error) {
	res = make([]byte, 1024)
	res, err = ioutil.ReadAll(conn)
	if err != nil {
		err = fmt.Errorf("Error whule receiving the data: %s", err.Error())
	}
	return
}

func SendTCPPacket(addr string, packet []byte) (res []byte, err error) {
	conn, err := connectTCPServer(addr)
	if err != nil {
		log.Error(err)
		return
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	if err != nil {
		err = fmt.Errorf("Error while sending the data: %s", err.Error())
		log.Error(err)
		return
	}

	res, err = readFromConn(conn)
	if err != nil {
		err = fmt.Errorf("Error while sending the data: %s", err.Error())
		log.Error(err)
	}
	return
}
