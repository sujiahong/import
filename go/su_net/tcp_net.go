
package su_net

import (
	"fmt"
	"net"
)




/////创建tcp服务器
func CreateTcpServer(addr string){
	tcpaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}
	listenconn, err := net.ListenTCP("tcp", tcpaddr)
	if err != nil {
		return
	}
	for {
		tcpconn, err := listenconn.AccpetTCP()
		if err != nil {
			continue
		}
	}
}
////创建客户端
func CreateTcpClient(addr string){
	tcpaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}
	tcpconn, err := net.DialTCP("tcp", tcpaddr)
	if err != nil {
		return
	}
	defer tcpconn.Close()

}