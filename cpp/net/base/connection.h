
////////////////
/// tcp连接封装
////////////////

#ifndef _CONNECTION_H_
#define _CONNECTION_H_

#include <string>

#include "../../toolbox/original_dependence.hpp"

namespace su 
{
class Channel;
class EventLoop;
class Socket;
class Connection: public Noncopyable, public std::enable_shared_from_this<Connection>
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

    CONNECTION_CALLBACK_TYPE m_connection_callback_;
    MESSAGE_CALLBACK_TYPE m_message_callback_;
    WRITE_COMPLETE_CALLBACK_TYPE m_write_complete_callback_;
    // HighWaterMarkCallback highWaterMarkCallback_;
    CLOSE_CALLBACK_TYPE m_close_callback_;
public:
    Connection(EventLoop* loop, 
        int a_fd, 
        const std::string& a_name, 
        std::string& a_ip, unsigned short a_port, 
        std::string& a_peer_ip, unsigned short a_peer_port);
    ~Connection();
public:
    int GetConnStat() const {return m_conn_stat_;}
    const std::string& GetName() const { return m_name_;}
    void GetLocalIPAndPort(std::string& a_ip, unsigned short& a_port)
    {
        a_ip = m_local_ip_;
        a_port = m_local_port_;
    }
    void GetPeerIPAndPort(std::string& a_ip, unsigned short& a_port)
    {
        a_ip = m_peer_ip_;
        a_port = m_peer_port_;
    }
    void SetTcpNoDelay(bool a_on);
    void Shutdown();
    void Close();
    void ForceClose();

    void ConnectionEstablished();
    void ConnectionDestroyed();

    void Send(const std::string& a_msg);

    void SetConnectionCallback(const CONNECTION_CALLBACK_TYPE& cb)
    { m_connection_callback_ = cb; }

    void SetMessageCallback(const MESSAGE_CALLBACK_TYPE& cb)
    { m_message_callback_ = cb; }

    void SetWriteCompleteCallback(const WRITE_COMPLETE_CALLBACK_TYPE& cb)
    { m_write_complete_callback_ = cb; }

    void SetCloseCallback(const CLOSE_CALLBACK_TYPE& cb)
    { m_close_callback_ = cb; }
private:
    void HandleRead(unsigned int a_rt_time);
    void HandleWrite();
    void HandleClose();
    void HandleError();
};

}//////namespace su
#endif