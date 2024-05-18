
////////////////
/// tcp连接封装
////////////////

#ifndef _CONNECTION_HPP_
#define _CONNECTION_HPP_

#include <vector>
#include <unordered_map>
#include <unordered_set>
#include <string>
#include <unistd.h>
#include <arpa/inet.h>
#include <sys/socket.h>

#include "../toolbox/original_dependence.hpp"
#include "socket.h"

namespace su {
class Connection: public Noncopyable
{
private:
    int m_fd_;/////连接的文件描述符
    std::string m_ip_;
    unsigned short m_port_;
    std::string m_peer_ip_;/////////对方ip
    unsigned short m_peer_port_;////对方port
    int m_conn_stat_;//////1 connecting,2 conected,3 disconneting,4 disconnected
    Buffer* m_inbuffer_;
    Buffer* m_outBuffer_;
public:
    Connection(int a_fd)
    {}
    Connection(int a_fd, std::string& a_ip, unsigned short a_port)
    {
        m_fd_ = a_fd;
        m_peer_ip_ = a_ip;
        m_peer_port_ = a_port;
    }
    Connection(std::string& a_ip, unsigned short a_port)
    {
        m_peer_ip_ = a_ip;
        m_peer_port_ = a_port;
        m_fd_ = socket(AF_INET, SOCK_STREAM, 0);
    }
    ~Connection()
    {}
public:
    


};
typedef std::shared_ptr<TcpConnection> TcpConnectionPtr;
}
#endif