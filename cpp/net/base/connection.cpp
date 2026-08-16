
#include <unistd.h>

#include "connection.h"
#include "event_loop.hpp"
#include "socket.h"

using namespace su;

Connection::Connection(EventLoop* a_loop, int a_fd, const std::string& a_name, 
    const std::string& a_ip, unsigned short a_port, 
    const std::string& a_peer_ip, unsigned short a_peer_port):m_loop_(a_loop), m_name_(a_name), m_conn_stat_(SE_CONNECTING),
        m_ip_(a_ip),m_port_(a_port), m_peer_ip_(a_peer_ip),m_peer_port_(a_peer_port)
{
    m_sock_.reset(new Socket(a_fd));
    m_channel_.reset(new Channel(m_loop_, m_sock_->Fd()));
    m_channel_->SetReadCallback(std::bind(&Connection::HandleRead, this, std::placeholders::_1));
    m_channel_->SetWriteCallback(std::bind(&Connection::HandleWrite, this));
    m_channel_->SetCloseCallback(std::bind(&Connection::HandleClose, this));
    m_channel_->SetErrorCallback(std::bind(&Connection::HandleError, this));
    ///// todo log

    m_sock_->SetKeepAlive(true);
}

Connection::~Connection()
{
    // m_sock_ / m_channel_ 由 unique_ptr 自动释放
}

int Connection::Fd() const
{
    return m_sock_ ? m_sock_->Fd() : -1;
}

void Connection::ConnectionEstablished()
{
    m_conn_stat_ = SE_CONNECTED;
    m_channel_->EnableRead();   ////开始监听读事件
    if (m_connection_callback_)
        m_connection_callback_(shared_from_this());
    ////// todo log
}

void Connection::ConnectionDestroyed()
{
    ////此时已在事件循环末尾的 pending functor 中执行，Channel 回调栈已返回，可安全移除
    if (m_conn_stat_ == SE_CONNECTED)
    {
        m_conn_stat_ = SE_DISCONNECTED;
        m_channel_->DisableAll();
    }
    m_loop_->RemoveChannel(m_channel_.get());
    ////// todo log
}

void Connection::SetTcpNoDelay(bool a_on)
{
    m_sock_->SetTcpNoDelay(a_on);
}

void Connection::Send(const std::string& a_msg)
{
    if (m_conn_stat_ != SE_CONNECTED)
        return;

    if (m_output_buffer_.ReadableBytes() == 0)
    {
        ////输出缓冲为空，尝试直接发送
        ssize_t n = ::write(m_sock_->Fd(), a_msg.data(), a_msg.size());
        if (n >= 0)
        {
            if (static_cast<size_t>(n) == a_msg.size())
            {
                ////一次写完
                if (m_write_complete_callback_)
                    m_write_complete_callback_(shared_from_this());
                return;
            }
            ////部分发送，剩余入缓冲，等待可写
            m_output_buffer_.Append(a_msg.data() + n, a_msg.size() - n);
        }
        else
        {
            ////EAGAIN/EWOULDBLOCK 等，全部入缓冲
            m_output_buffer_.Append(a_msg.data(), a_msg.size());
        }
        m_channel_->EnableWrite();
    }
    else
    {
        ////输出缓冲已有未发数据，直接追加，等待可写事件
        m_output_buffer_.Append(a_msg.data(), a_msg.size());
    }
}

void Connection::Shutdown()
{
    if (m_conn_stat_ == SE_CONNECTED)
    {
        m_conn_stat_ = SE_DISCONNECTING;
        m_sock_->ShutdownWrite();
    }
}

void Connection::Close()
{
    if (m_conn_stat_ == SE_CONNECTED || m_conn_stat_ == SE_DISCONNECTING)
    {
        m_conn_stat_ = SE_DISCONNECTING;
        ////无待发数据直接关闭，否则等写完后关闭
        if (m_output_buffer_.ReadableBytes() == 0)
            HandleClose();
        else
            m_channel_->EnableWrite();
    }
}

void Connection::ForceClose()
{
    HandleClose();
}

void Connection::HandleRead(unsigned int a_rt_time)
{
    int saved_errno = 0;
    ssize_t n = m_input_buffer_.ReadFd(m_sock_->Fd(), &saved_errno);
    if (n > 0)
    {
        ////把读到的数据交给上层处理
        if (m_message_callback_)
            m_message_callback_(shared_from_this(), &m_input_buffer_, a_rt_time);
    }
    else if (n == 0)
    {
        HandleClose();
    }
    else
    {
        if (saved_errno == EAGAIN || saved_errno == EWOULDBLOCK)
            return;
        HandleError();
    }
}

void Connection::HandleWrite()
{
    if (!m_channel_->IsWriting())
        return;

    ssize_t n = ::write(m_sock_->Fd(), m_output_buffer_.Peek(), m_output_buffer_.ReadableBytes());
    if (n >= 0)
    {
        m_output_buffer_.Retrieve(n);
        if (m_output_buffer_.ReadableBytes() == 0)
        {
            m_channel_->DisableWrite();
            if (m_write_complete_callback_)
                m_write_complete_callback_(shared_from_this());
            ////写完后若处于半关闭流程则完成关闭
            if (m_conn_stat_ == SE_DISCONNECTING)
                Shutdown();
        }
    }
    else
    {
        HandleError();
    }
}

void Connection::HandleClose()
{
    if (m_conn_stat_ == SE_DISCONNECTED)
        return;
    m_conn_stat_ = SE_DISCONNECTED;
    m_channel_->DisableAll();
    ////// todo log
    if (m_close_callback_)
    {
        TCP_CONNECTION_PTR guard = shared_from_this();
        m_close_callback_(guard);
    }
    ////注意：guard 析构后本对象可能已销毁，此后不得再访问成员
}

void Connection::HandleError()
{
    ////// todo log
    HandleClose();
}
