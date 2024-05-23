
#include "connection.h"
#include "event_loop.hpp"
#include "socket.h"

using namespace su;

Connection::Connection(EventLoop* a_loop, int a_fd, const std::string& a_name, 
    std::string& a_ip, unsigned short a_port, 
    std::string& a_peer_ip, unsigned short a_peer_port):m_loop_(a_loop), m_name_(a_name), m_conn_stat_(SE_CONNECTING),
        m_ip_(a_ip),m_port_(a_port), m_peer_ip_(a_peer_ip),m_peer_port_(a_peer_port)
{
    m_sock_ = new Socket(a_fd);
    m_channel_ = new Channel(m_loop_, m_sock_->Fd());
    m_channel_->SetReadCallback(std::bind(&Connection::HandleRead, this, std::placeholders::_1));
    m_channel_->SetWriteCallback(std::bind(&Connection::HandleWrite, this));
    m_channel_->SetCloseCallback(std::bind(&Connection::HandleClose, this));
    m_channel_->SetErrorCallback(std::bind(&Connection::HandleError, this));
    ///// todo log

    m_sock_->SetKeepAlive(true);
}

Connection::~Connection()
{
    // delete m_sock_;
    // delete m_channel_;
}

void Connection::ConnectionEstablished()
{
    m_conn_stat_ = SE_CONNECTED;
    ////// todo log
}

void Connection::ConnectionDestroyed()
{
    m_loop_->RemoveChannel(m_channel_);
    ////// todo log
}
void Connection::SetTcpNoDelay(bool a_on)
{
    m_sock_->SetTcpNoDelay(a_on);
}
void Connection::Shutdown() ////暂时不支持
{
}
void Connection::ForceClose()////暂时不支持
{
}
void Connection::HandleRead(unsigned int a_rt_time)
{

}

void Connection::HandleWrite()
{

}

void Connection::HandleClose()
{
    m_conn_stat_ = SE_CLOSED;
    ////// todo log
    m_loop_->RemoveChannel(m_channel_);
    ////// todo log
}

void Connection::HandleError()
{
    ////// todo log
    m_loop_->RemoveChannel(m_channel_);
    ////// todo log
}
