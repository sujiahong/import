////////
///io复用
////////

#ifndef _POLLER_HPP_
#define _POLLER_HPP_

#include <vector>
#include <unordered_map>

#include "../toolbox/original_dependence.hpp"
#include "event_loop.hpp"

namespace su
{
class PollerBase: Noncopyable
{
private:
    EventLoop* m_owner_loop_;
    std::unordered_map<int, Channel*> m_channels_;
public:
    PollerBase(EventLoop* a_loop_ptr)
    {

    }
    virtual ~PollerBase()
    {

    }

public:
    virtual unsigned long long Poll(int a_timeout, std::vector<Channel*>& a_active_channels)=0;
    virtual void UpdateChannel(Channel* a_channel)=0;
    //virtual void RemoveChannel(Channel* a_channel)=0;
};

class Poll: public PollerBase
{
private:
    /* data */
public:
    Poll(/* args */)
    {

    }
    ~Poll()
    {

    }
};

class EPoll: PollerBase
{
private:
    /* data */
public:
    EPoll(/* args */)
    {

    }
    ~EPoll()
    {

    }
};


}


#endif