///////
///
////////

#ifndef _CHANNEL_HPP_
#define _CHANNEL_HPP_

#include "../toolbox/original_dependence.hpp"
#include <functional>

namespace su
{

class EventLoop;

class Channel: Noncopyable
{
    typedef 
private:
    EventLoop* m_loop_;
    int m_fd_;
public:
    Channel(EventLoop* a_loop, int a_fd):m_loop_(a_loop),m_fd_(a_fd)
    {}
    ~Channel()
    {}


};


}


#endif