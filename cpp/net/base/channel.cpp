
#include <sstream>

#include "channel.h"
#include "event_loop.hpp"

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
    if (m_read_callback_ & EPOLLERR) // error
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