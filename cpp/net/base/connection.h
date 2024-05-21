
////////////////
/// tcp连接封装
////////////////

#ifndef _CONNECTION_HPP_
#define _CONNECTION_HPP_

#include <string>


#include "../toolbox/original_dependence.hpp"

namespace su {
class Channel;
class EventLoop;
class Socket;
class Connection: public Noncopyable, std::enable_shared_from_this<Connection>
{
    enum StateEnum
    {
        SE_CONNECTING,
        SE_CONNECTED,
        SE_DISCONNECTING,
        SE_DISCONNECTED
    };
private:
    const std::string m_name_;
    std::string m_ip_;
    unsigned short m_port_;
    std::string m_peer_ip_;/////////对方ip
    unsigned short m_peer_port_;////对方port

    int m_conn_stat_;

    EventLoop* m_loop_;
    Socket* m_sock_;
    Channel* m_channel_;


public:
    Connection(EventLoop* loop, 
        int a_fd, 
        const std::string& a_name, 
        std::string& a_ip, unsigned short a_port, 
        std::string& a_peer_ip, unsigned short a_peer_port);
    ~Connection();
public:
    int GetConnStat();

    void SetTcpNoDelay(bool a_on);

    void ConnectionEstablished();
    void ConnectionDestroyed();
    void Read(const std::string& a_msg);
    void Write(const std::string& a_msg);

private:
    void HandleRead(unsigned int a_rt_time);
    void HandleWrite();
    void HandleClose();
    void HandleError();
};

}
#endif