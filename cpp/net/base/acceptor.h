///////////////
//////连接接收器
///////////////

#ifndef ACCEPTOR_H
#define ACCEPTOR_H

#include <vector>
#include <string>
#include <sys/types.h>

#include "base_define.h"
#include "../../toolbox/original_dependence.hpp"
#include "socket.h"
#include "channel.h"

namespace su {
class EventLoop;
class Acceptor: public Noncopyable 
{   
private:
    unsigned short m_ltn_port_;      //监听端口
    std::string m_ltn_ip_;          // 监听ip
    Socket m_listen_sock_;          //监听socket
    Channel m_listen_channel_;
    NEW_CONNECTION_CALLBACK_TYPE m_new_connection_callback_;
public:
    Acceptor(EventLoop* a_loop, const std::string& a_ip_port, bool a_reuse_port):
        m_listen_sock_(AF_INET, SOCK_STREAM | SOCK_NONBLOCK | SOCK_CLOEXEC),
        m_listen_channel_(a_loop, &m_listen_sock_)
    {
        m_listen_sock_.SetReuseAddr(true);
        m_listen_sock_.SetReusePort(a_reuse_port);
        std::vector<std::string> str_vec;
        su::Split(a_ip_port, ":", str_vec);
        if (str_vec.size() == 1)
        {
            m_ltn_ip_="";
            m_ltn_port_ = su::String2Number<unsigned short>(str_vec[0]);
        }
        else if (str_vec.szie() == 2)
        {
            m_ltn_ip_ = str_vec[0];
            m_ltn_port_ = su::String2Number<unsigned short>(str_vec[1]);
        }
        else
        {
            //log
        }
        m_listen_sock_.Bind(m_ltn_ip_, m_ltn_port_);
        m_listen_sock_.Listen();
        m_listen_channel_.SetReadCallback(std::bind(&Acceptor::AcceptConnection, this));
        m_listen_channel_.EnableRead();
    }
    ~Acceptor()
    {
        m_listen_channel_.DisableAll();
    }
    void SetNewConnectionCallback(NewConnectionCallback a_cb)
    {
        m_new_connection_callback_ = a_cb;
    }
private:
    void AcceptConnection()
    {
        std::string peer_ip;
        unsigned short peer_port;
        int conn_fd = m_listen_sock_.Accept(peer_ip, peer_port);
        if (conn_fd >= 0)
        {
            if (m_new_connection_callback_)
            {
                m_new_connection_callback_(conn_fd, peer_ip, peer_port);
            }
            else
                ::close(conn_fd);
        } 
        else {

        }
    }
};
}//// namespace su
#endif