///////////////
//////连接接收器
///////////////

#ifndef ACCEPTOR_H
#define ACCEPTOR_H


#include "../../toolbox/original_dependence.hpp"
#include "socket.h"
#include "channel.h"
namespace su {

class Acceptor: public Noncopyable 
{
public:
    typedef std::function<void(int, std::string, unsigned short)> NewConnectionCallback;
private:
    Socket m_listen_sock_;
    bool m_listening_;
    Channel m_listen_channel_;
    NewConnectionCallback m_new_connection_callback_;
public:
    Acceptor(const std::string& a_ip, const unsigned short a_port, bool a_reuse_port):
        m_listen_sock_(AF_INET, SOCK_STREAM),
        m_listening_(false),
        m_listen_channel_(NULL, m_listen_sock_.Fd())
    {
        m_listen_sock_.SetReuseAddr(true);
        m_listen_sock_.SetReusePort(a_reuse_port);
        m_listen_sock_.Bind(a_ip, a_port);
        m_listen_channel_.SetReadCallback(std::bind(&Acceptor::handleRead, this));
    }
    ~Acceptor()
    {
        m_listen_channel_.DisableAll();
    }

    void listen() 
    {
        m_listening_ = true;
        m_listen_sock_.Listen();
        m_listen_channel_.EnableReading();
    }
    bool listening() {return m_listening_;}
private:
    void HandleRead()
    {

    }
}
}
#endif