////////
/////
///////
#ifndef _EVENT_LOOP_HPP_
#define _EVENT_LOOP_HPP_

#include <pthread.h>
#include "../toolbox/original_dependence.hpp"

namespace su
{

class EventLoop: Noncopyable
{
private:
    bool m_loop_;
    const unsigned int m_thread_id_;
public:
    EventLoop():m_loop_(false),m_thread_id_(pthread_self())
    {}
    ~EventLoop();
    {
    }
public:
    void Loop()
    {
        assert(!m_loop_)

    }
    bool IsInLoopThread()
    {
        return (m_thread_id_ == pthread_self());
    }
    void AssertInLoopThread()
    {} 
};

__thread EventLoop* t_loopInThisThread = NULL;


}


#endif