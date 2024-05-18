/////////////////
//////// channel 封装
////////////////

#ifndef _CHANNEL_H_
#define _CHANNEL_H_

#include <functional>
#include <sys/epoll.h>
#include "define_struct.h"
#include "../../toolbox/original_dependence.hpp"
#include "socket.h"
#include "event_loop.h"
namespace su
{

const int NONE_EVENT = 0;
const int READ_EVENT = EPOLLIN | EPOLLPRI;
const int WRITE_EVENT = EPOLLOUT;
class Channel: public Noncopyable
{
private:
    EventLoop* m_loop_;
    Socket* m_sock_;
    uint32_t m_events_;
    uint32_t m_ready_events_;
    int m_index_;
    bool m_is_in_epoll_;

    EVENT_CALLBACK_TYPE m_write_callback_;
    EVENT_CALLBACK_TYPE m_close_callback_;
    EVENT_CALLBACK_TYPE m_error_callback_;
    READ_EVENT_CALLBACK_TYPE m_read_callback_;
public:
    Channel(EventLoop* a_loop, Socket* a_sock):m_loop_(a_loop),m_sock_(a_sock)
    {}
    ~Channel()
    {}
public:
    inline int Fd() const{ return m_sock_.Fd();}
    inline int Index() { return m_index_; }
    inline void SetIndex(int idx) { m_index_ = idx; }
    inline uint32_t Events() const {return m_events_;}
    inline bool GetInEpoll() const {return m_is_in_epoll_;}
    inline void SetInEpoll() { m_is_in_epoll_ = true;}
    inline void SetReadyEvents(uint32_t a_evts) { m_ready_events_= a_evts;}
    
    inline bool IsNoneEvent() const {return m_events_ == NONE_EVENT;}
    inline bool IsWriting() const { return m_events_ & WRITE_EVENT; }
    inline bool IsReading() const { return m_events_ & READ_EVENT; }

    inline void EnableRead() { m_events_ |= READ_EVENT; Update(); }
    inline void DisableRead() { m_events_ &= ~READ_EVENT; Update(); }
    inline void EnableWrite() { m_events_ |= WRITE_EVENT; Update(); }
    inline void DisableWrite() { m_events_ &= ~WRITE_EVENT; Update(); }
    inline void DisableAll() { m_events_ = NONE_EVENT; Update(); }

    inline void SetWriteCallback(EVENT_CALLBACK_TYPE a_cb)
    { m_write_callback_ = a_cb; }
    inline void SetCloseCallback(EVENT_CALLBACK_TYPE a_cb)
    { m_close_callback_ = a_cb; }
    inline void SetErrorCallback(EVENT_CALLBACK_TYPE a_cb)
    { m_error_callback_ = a_cb; }
    inline void SetReadCallback(READ_EVENT_CALLBACK_TYPE a_cb)
    { m_read_callback_ = a_cb; }

    void HandleEvent(uint32_t a_rt_time)
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

}/////namespace su

#endif