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
const int NONE_EVENT = 0;
const int READ_EVENT = EPOLLIN | EPOLLPRI;
const int WRITE_EVENT = EPOLLOUT;
class Channel: public Noncopyable
{
    typedef std::function<void()> EVENT_CALLBACK_TYPE;
    typedef std::function<void(unsigned int)> READ_EVENT_CALLBACK_TYPE;
private:
    int fd_;
    int m_events_;
    int m_index_;

    EVENT_CALLBACK_TYPE m_write_callback_;
    EVENT_CALLBACK_TYPE m_close_callback_;
    EVENT_CALLBACK_TYPE m_error_callback_;
    READ_EVENT_CALLBACK_TYPE m_read_callback_;
public:
    Channel(int a_fd):fd_(a_fd)
    {}
    ~Channel()
    {}
public:
    inline int Fd() const{ return fd_;}
    inline int Index() { return m_index_; }
    inline void SetIndex(int idx) { m_index_ = idx; }
    inline int Events() const {return m_events_;}
    
    inline bool IsNoneEvent() const {return m_events_ == NONE_EVENT;}
    inline bool IsWriting() const { return m_events_ & WRITE_EVENT; }
    inline bool IsReading() const { return m_events_ & READ_EVENT; }
    inline void EnableReading() { m_events_ |= READ_EVENT; update(); }
    inline void DisableReading() { m_events_ &= ~READ_EVENT; update(); }
    inline void EnableWriting() { m_events_ |= WRITE_EVENT; update(); }
    inline void DisableWriting() { m_events_ &= ~WRITE_EVENT; update(); }
    inline void DisableAll() { m_events_ = NONE_EVENT; update(); }
};

}

#endif