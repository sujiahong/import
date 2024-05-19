
#ifndef _TCP_SERVER_H_
#define _TCP_SERVER_H_

#include <string>
#include <unordered_map>

#include "../../toolbox/original_dependence.hpp"

#include "tcp_connection.hpp"

namespace su
{
class EventLoop;
class Acceptor;
class TcpServer: public Noncopyable
{
private:
    const std::string m_ip_port_; ///ip端口串
    EventLoop* m_loop_;
    Acceptor m_acceptor_;
    unsigned int m_conn_id;////连接id
    typedef std::unordered_map<unsigned int, su::TcpConnectionPtr> CONNECTION_MAP_TYPE;
    CONNECTION_MAP_TYPE m_connections_;//////所有连接

public:
    TcpServer(EventLoop* a_loop, const std::string& a_ip_port);
    ~TcpServer();

    void Run();
    void SetThreadNum(int a_num);

    void NewConnection(int a_fd, std::string a_peer_id, unsigned short a_peer_port);
    void RemoveConnection(unsigned int a_conn_id);

};


}
#endif