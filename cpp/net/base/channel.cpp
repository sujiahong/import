
#include <sstream>

#include "channel.h"
#include "event_loop.hpp"

/*
EPOLLIN      // 可读：socket 有数据可读，listen fd 有新连接可 accept
EPOLLOUT     // 可写：发送缓冲区可写
EPOLLPRI     // 有紧急数据，例如 TCP out-of-band 数据
EPOLLERR     // 错误事件
EPOLLHUP     // 挂起/对端关闭
EPOLLRDHUP   // 对端关闭写半边，TCP half-close
EPOLLET      // 边缘触发 Edge Triggered
EPOLLONESHOT // 事件触发一次后自动禁用，需要重新 arm
EPOLLEXCLUSIVE // 多 epoll 等同一个 fd 时减少惊群，Linux 4.5+
EPOLLWAKEUP  // 防止系统 suspend，通常需要权限
*/

su::Channel::Channel(EventLoop* a_loop, int a_fd):m_loop_(a_loop),m_fd_(a_fd)
{
}

su::Channel::~Channel()
{
}

void su::Channel::HandleEvent(unsigned int a_rt_time)
{
    if ((m_ready_events_ & EPOLLHUP) && !(m_ready_events_& EPOLLIN)) //关闭
    {
        if (m_close_callback_) m_close_callback_();
    }
    if (m_ready_events_ & EPOLLERR) // error
    {
        if (m_error_callback_) m_error_callback_();
    }
    if (m_ready_events_ & (EPOLLIN | EPOLLPRI | EPOLLRDHUP))//读
    {
        if (m_read_callback_) m_read_callback_(a_rt_time);
    }
    if (m_ready_events_ & EPOLLOUT) /// 写
    {
        if (m_write_callback_) m_write_callback_();
    }
}

void su::Channel::Update()
{
    m_loop_->UpdateChannel(this);
}

std::string su::Channel::EventsToString(int fd, int ev)
{
    std::ostringstream oss;
    oss << fd << ": ";
    if (ev & EPOLLIN)
      oss << "IN ";
    if (ev & EPOLLPRI)
      oss << "PRI ";
    if (ev & EPOLLOUT)
      oss << "OUT ";
    if (ev & EPOLLHUP)
      oss << "HUP ";
    if (ev & EPOLLRDHUP)
      oss << "RDHUP ";
    if (ev & EPOLLERR)
      oss << "ERR ";
    return oss.str();
}
