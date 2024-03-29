
#ifndef _TCP_SERVER_H_
#define _TCP_SERVER_H_

#include <string>
#include <unordered_map>

#include "../toolbox/original_dependence.hpp"
#include "tcp_connection.hpp"

namespace su
{

class TcpServer: public Noncopyable
{
private:
    const std::string m_ip_port_; ///ip端口串
    int m_ltn_port_;      //监听端口
    std::string m_ltn_ip_;// 监听ip

    unsigned int m_conn_id;////连接id
    typedef std::unordered_map<unsigned int, su::TcpConnectionPtr> CONNECTION_MAP_TYPE;
    CONNECTION_MAP_TYPE m_connections_;//////所有连接

public:
    TcpServer(const std::string& a_ip_port);
    ~TcpServer();

    void Launch();

};


}
#endif