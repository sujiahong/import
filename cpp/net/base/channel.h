/////////////////
//////// channel 封装
////////////////

#ifndef _CHANNEL_H_
#define _CHANNEL_H_

#include <functional>
#include <sys/epoll.h>
#include "../../toolbox/original_dependence.hpp"
#include "socket.h"
namespace su
{

class EventLoop;
const int NONE_EVENT = 0;
const int READ_EVENT = EPOLLIN | EPOLLPRI;
const int WRITE_EVENT = EPOLLOUT;
class Channel: public Noncopyable
{
    typedef std::function<void()> EVENT_CALLBACK_TYPE;
    typedef std::function<void(unsigned int)> READ_EVENT_CALLBACK_TYPE;
private:
    EventLoop* m_loop_;
    int fd_;
    uint32_t m_events_;
    uint32_t m_ready_events_;
    int m_index_;
    bool m_is_in_epoll_;

    EVENT_CALLBACK_TYPE m_write_callback_;
    EVENT_CALLBACK_TYPE m_close_callback_;
    EVENT_CALLBACK_TYPE m_error_callback_;
    READ_EVENT_CALLBACK_TYPE m_read_callback_;
public:
    Channel(EventLoop* a_loop, int a_fd):m_loop_(a_loop),fd_(a_fd)
    {}
    ~Channel()
    {}
public:
    inline int Fd() const{ return fd_;}
    inline int Index() { return m_index_; }
    inline void SetIndex(int idx) { m_index_ = idx; }
    inline uint32_t Events() const {return m_events_;}
    inline bool GetInEpoll() const {return m_is_in_epoll_;}
    inline void SetInEpoll() { m_is_in_epoll_ = true;}
    inline void SetReadyEvents(uint32_t a_evts) { m_ready_events_= a_evts;}
    
    inline bool IsNoneEvent() const {return m_events_ == NONE_EVENT;}
    inline bool IsWriting() const { return m_events_ & WRITE_EVENT; }
    inline bool IsReading() const { return m_events_ & READ_EVENT; }

    inline void EnableReading() { m_events_ |= READ_EVENT; Update(); }
    inline void DisableReading() { m_events_ &= ~READ_EVENT; Update(); }
    inline void EnableWriting() { m_events_ |= WRITE_EVENT; Update(); }
    inline void DisableWriting() { m_events_ &= ~WRITE_EVENT; Update(); }
    inline void DisableAll() { m_events_ = NONE_EVENT; Update(); }


    void HandleEvent(uint32_t a_rt_time)
    {
        if ((m_ready_events_ & EPOLLHUP) && !(m_ready_events_& EPOLLIN)) 
        {
            if (m_close_callback_) m_close_callback_();
        }
        if (m_read_callback_ & EPOLLERR)
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
private:
    void Update()
    {
        m_loop_->UpdateChannel(this);
    }
    std::string EventsToString(int fd, int ev)
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
};

}

#endif