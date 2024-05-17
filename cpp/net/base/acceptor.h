///////////////
//////连接接收器
///////////////

#ifndef ACCEPTOR_H
#define ACCEPTOR_H

#include <vector>
#include <string>
#include "../../toolbox/original_dependence.hpp"
#include "socket.h"
#include "channel.h"
namespace su {
class EventLoop;
class Acceptor: public Noncopyable 
{
public:
    typedef std::function<void(int, std::string, unsigned short)> NewConnectionCallback;
private:
    int m_ltn_port_;      //监听端口
    std::string m_ltn_ip_;// 监听ip
    Socket m_listen_sock_; //监听socket
    bool m_listening_;    //监听状态
    Channel m_listen_channel_;
    NewConnectionCallback m_new_connection_callback_;
public:
    Acceptor(EventLoop* a_loop, const std::string& a_ip_port, bool a_reuse_port):
        m_listen_sock_(AF_INET, SOCK_STREAM),
        m_listening_(false),
        m_listen_channel_(a_loop, m_listen_sock_.Fd())
    {
        m_listen_sock_.SetReuseAddr(true);
        m_listen_sock_.SetReusePort(a_reuse_port);
        std::vector<std::string> str_vec;
        su::Split(a_ip_port, ":", str_vec);
        if (str_vec.size() == 1)
        {
            m_ltn_ip_="";
            m_ltn_port_ = su::String2Number<int>(str_vec[0]);
        }
        else if (str_vec.szie() == 2)
        {
            m_ltn_ip_ = str_vec[0];
            m_ltn_port_ = su::String2Number<int>(str_vec[1]);
        }
        else
        {
            //log
        }
        m_listen_sock_.Bind(m_ltn_ip_, m_ltn_port_);
        m_listen_channel_.SetReadCallback(std::bind(&Acceptor::HandleRead, this));
    }
    ~Acceptor()
    {
        m_listen_channel_.DisableAll();
    }

    void Listen() 
    {
        m_listening_ = true;
        m_listen_sock_.Listen();
        m_listen_channel_.EnableReading();
    }
    bool Listening() {return m_listening_;}
private:
    void HandleRead()
    {

    }
}
}
#endif