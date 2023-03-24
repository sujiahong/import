////////////////
/// tcp连接
////////////////

#ifndef _TCP_CONNECTION_HPP_
#define _TCP_CONNECTION_HPP_

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
class TcpConnection: public Noncopyable
{
private:
    int m_fd_;/////连接的文件描述符
    std::string m_ip_;
    unsigned short m_port_;
    std::string m_peer_ip_;/////////对方ip
    unsigned short m_peer_port_;////对方port
public:
    TcpConnection(int a_fd)
    {}
    TcpConnection(int a_fd, std::string& a_ip, unsigned short a_port)
    {
        m_fd_ = a_fd;
        m_peer_ip_ = a_ip;
        m_peer_port_ = a_port;
    }
    TcpConnection(std::string& a_ip, unsigned short a_port)
    {
        m_peer_ip_ = a_ip;
        m_peer_port_ = a_port;
        m_fd_ = socket(AF_INET, SOCK_STREAM, 0);
    }
    ~TcpConnection()
    {}
public:
    int Connect()
    {
        struct sockaddr_in peer_addr;
        peer_addr.sin_family = AF_INET;////IPv4地址
        peer_addr.sin_addr.s_addr = inet_addr(m_peer_ip_.c_str());
        peer_addr.sin_port = htons(m_peer_port_);
        if (m_fd_ > 0)
        {
            int ret = connect(m_fd_, (struct sockaddr*)&peer_addr, sizeof(peer_addr));
            if (ret != 0)
            {
                ////打印日志
                return ret;
            }
        }
        else
        {
            ////打印日志
            return -2;
        }
        return 0;
    }
    int Reconnect(std::string& a_ip, unsigned short a_port)
    {
        m_peer_ip_ = a_ip;
        m_peer_port_ = a_port;
        m_fd_ = socket(AF_INET, SOCK_STREAM, 0);
        if (m_fd_ > 0)
            Connect();
        else
        {
            /////打印日志
            return -2;
        }
        return 0;
    }
    int Bind(int a_fd,  std::string& a_ip, unsigned short a_port)
    {
        m_fd_ = a_fd;
        m_peer_ip_ = a_ip;
        m_peer_port_ = a_port;
    }
    int Listen()
    {

    }


};
}
#endif