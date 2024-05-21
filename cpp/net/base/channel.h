/////////////////
//////// channel 封装
////////////////

#ifndef _CHANNEL_H_
#define _CHANNEL_H_

#include <functional>
#include <string>
#include <sys/epoll.h>
#include "define_struct.h"
#include "../../toolbox/original_dependence.hpp"


namespace su
{
class EventLoop;
const int NONE_EVENT = 0;
const int READ_EVENT = EPOLLIN | EPOLLPRI;
const int WRITE_EVENT = EPOLLOUT;
class Channel: public Noncopyable
{
private:
    EventLoop* m_loop_;
    int m_fd_;
    int m_events_;
    int m_ready_events_;
    int m_index_;
    bool m_is_in_epoll_;

    EVENT_CALLBACK_TYPE m_write_callback_;
    EVENT_CALLBACK_TYPE m_close_callback_;
    EVENT_CALLBACK_TYPE m_error_callback_;
    READ_EVENT_CALLBACK_TYPE m_read_callback_;
public:
    Channel(EventLoop* a_loop, int a_fd);
    ~Channel();
private:
    void Update();
    std::string EventsToString(int fd, int ev);
public:
    inline int Fd() const{ return m_fd_;}
    inline int Index() { return m_index_; }
    inline void SetIndex(int idx) { m_index_ = idx; }
    inline int Events() const {return m_events_;}
    inline bool GetInEpoll() const {return m_is_in_epoll_;}
    inline void SetInEpoll() { m_is_in_epoll_ = true;}
    inline void SetReadyEvents(int a_evts) { m_ready_events_= a_evts;}
    
    inline bool IsNoneEvent() const {return m_events_ == NONE_EVENT;}
    inline bool IsWriting() const { return m_events_ & WRITE_EVENT; }
    inline bool IsReading() const { return m_events_ & READ_EVENT; }

    inline void EnableRead() { m_events_ |= READ_EVENT; Update(); }
    inline void DisableRead() { m_events_ &= ~READ_EVENT; Update(); }
    inline void EnableWrite() { m_events_ |= WRITE_EVENT; Update(); }
    inline void DisableWrite() { m_events_ &= ~WRITE_EVENT; Update(); }
    inline void DisableAll() { m_events_ = NONE_EVENT; Update(); }

    inline void SetWriteCallback(EVENT_CALLBACK_TYPE a_cb)
    { m_write_callback_ = std::move(a_cb); }
    inline void SetCloseCallback(EVENT_CALLBACK_TYPE a_cb)
    { m_close_callback_ = std::move(a_cb); }
    inline void SetErrorCallback(EVENT_CALLBACK_TYPE a_cb)
    { m_error_callback_ = std::move(a_cb); }
    inline void SetReadCallback(READ_EVENT_CALLBACK_TYPE a_cb)
    { m_read_callback_ = std::move(a_cb); }

    void HandleEvent(unsigned int a_rt_time);

};

}/////namespace su

#endif