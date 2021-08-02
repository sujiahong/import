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
class Channel;

class PollerBase: Noncopyable
{
public:
    typedef std::unordered_map<int, Channel*> ChannelHashType;
    typedef std::vector<Channel*> ChannelVecType;
private:
    
    EventLoop* m_owner_loop_;
    std::unordered_map<int, Channel*> m_channels_;
    typedef std::unordered_map<int, Channel*> ChannelMapType;
    ChannelHashType m_channels_;

public:
    PollerBase(EventLoop* a_loop_ptr):m_owner_loop_(a_loop_ptr)
    {}
    virtual ~PollerBase()
    {}

public:
    virtual unsigned long long Poll(int a_timeout, ChannelVecType& a_active_channels)=0;
    virtual void UpdateChannel(Channel* a_channel)=0;
    //virtual void RemoveChannel(Channel* a_channel)=0;
    virtual bool IsHaveChannel(Channel* a_channel) const;

    virtual bool IsExistChannel(Channel* a_channel_ptr);
    void AssertInLoopThread()
    {
        m_owner_loop_->AssertInLoopThread();
    }
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