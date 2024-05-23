
#ifndef _TCP_SERVER_H_
#define _TCP_SERVER_H_

#include <string>


#include "../../toolbox/original_dependence.hpp"
#include "../base/base_define.h"

namespace su
{
class EventLoop;
class Acceptor;
class EventLoopThreadPool;
class TcpServer: public Noncopyable
{
    
private:
    const std::string m_name_;
    const std::string m_ip_port_; ///ip端口串
    EventLoop* m_accept_loop_;
    Acceptor m_acceptor_;
    EventLoopThreadPool m_thread_pool_;
    unsigned int m_conn_id_;////连接id
    
    CONNECTION_MAP_TYPE m_connections_;//////所有连接

    CONNECTION_CALLBACK_TYPE m_connection_callback_;
    MESSAGE_CALLBACK_TYPE m_message_callback_;
    WRITE_COMPLETE_CALLBACK_TYPE m_write_complete_callback_;

    
public:
    TcpServer(EventLoop* a_loop, const std::string& a_ip_port, const std::string& a_name);
    ~TcpServer();

    void Run();
    void SetThreadNum(int a_num);

    inline void SetConnectionCallback(const CONNECTION_CALLBACK_TYPE a_cb)
    {
        m_connection_callback_ = a_cb;
    }
    inline void SetMessageCallback(const MESSAGE_CALLBACK_TYPE a_cb)
    {
        m_message_callback_ = a_cb;
    }
    inline void SetWriteCompleteCallback(const WRITE_COMPLETE_CALLBACK_TYPE a_cb)
    {
        m_write_complete_callback_ = a_cb;
    }
private:
    void NewConnection(int a_fd, const std::string& a_peer_id, unsigned short a_peer_port);
    void RemoveConnection(int a_fd);
};


}
#endif