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

/*
其中 events 成员描述事件类型，可以是以下几种类型宏的集合：

EPOLLIN：表示对应的文件描述符可以读（包括对端SOCKET正常关闭）；

EPOLLOUT：表示对应的文件描述符可以写；

EPOLLPRI：表示对应的文件描述符有紧急的数据可读（这里应该表示有带外数据到来）；

EPOLLERR：表示对应的文件描述符发生错误；

EPOLLHUP：表示对应的文件描述符被挂断；

EPOLLET： 将EPOLL设为边缘触发(Edge Triggered)模式，这是相对于水平触发(Level Triggered)来说的。

EPOLLONESHOT：只监听一次事件，当监听完这次事件之后，如果还需要继续监听这个socket的话，需要再次把这个socket加入到EPOLL队列里
*/

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