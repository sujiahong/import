////////////
////epoll封装
////////////

#ifndef _EPOOL_H_
#define _EPOLL_H_

#include <map>

#include <sys/epoll.h>
#include "../../toolbox/original_dependence.hpp"
#include "../../toolbox/time_function.hpp"
#include "channel.h"

namespace su {
/*
其中 events 成员描述事件类型，可以是以下几种类型宏的集合：

EPOLLIN：表示对应的文件描述符可以读（包括对端SOCKET正常关闭）；

EPOLLOUT：表示对应的文件描述符可以写；

EPOLLPRI：表示对应的文件描述符有紧急的数据可读（这里应该表示有带外数据到来）；

EPOLLERR：表示对应的文件描述符发生错误；

EPOLLHUP：表示对应的文件描述符被挂断；

EPOLLET： 将EPOLL设为边缘触发(Edge Triggered)模式，这是相对于水平触发(Level Triggered)来说的。

EPOLLONESHOT：只监听一次事件，当监听完这次事件之后，如果还需要继续监听这个socket的话，需要再次把这个socket加入到EPOLL队列里

typedef union epoll_data {
    void *ptr;
    int fd;
    __uint32_t u32;
    __uint64_t u64;
} epoll_data_t;

struct epoll_event {
    __uint32_t events; // Epoll events
    epoll_data_t data; // User data variable 
};
*/
#define MAX_EVENT_NUM 20000

class EPoll: public Noncopyable
{
private:
    int m_epfd_;
    struct epoll_event m_event_arr_[MAX_EVENT_NUM];
    typedef std::map<int, Channel*> CHANNEL_MAP_TYPE;
public:
    EPoll()
    {
        m_epfd_ = ::epoll_create1(0);
        assert(m_epfd_ > 0);
        memset(m_event_arr_, 0, sizeof(m_event_arr_)*MAX_EVENT_NUM)
    }
    ~EPoll()
    {
        ::close(m_epfd_);
    }
    const std::string operationToString(int op)
    {
        switch (op)
        {
          case EPOLL_CTL_ADD:
            return "ADD";
          case EPOLL_CTL_DEL:
            return "DEL";
          case EPOLL_CTL_MOD:
            return "MOD";
          default:
            assert(false && "ERROR op");
            return "Unknown Operation";
        }
    }
    unsigned int Poll(int a_tmout, std::vector<Channel*>& a_active_channels)
    {
        unsigned int now = (unsigned int)su::SecondTime();
        int numEvents = ::epoll_wait(m_epfd_, m_event_arr_, MAX_EVENT_NUM, a_tmout);
        if (numEvents < 0)
        {
            ///log
        }
        else 
        {
            assert(numEvents <= MAX_EVENT_NUM)
            //////log
            a_active_channels.reserve(numEvents);
            for (int i=0; i < numEvents; ++i)
            {
                Channel* ptr = static_cast<Channel*>(m_event_arr_[i].data.ptr);
                ptr->SetReadyEvents(m_event_arr_[i].events);
                a_active_channels.push_back(ptr);
            }
        }
        return now;
    }
    void RemoveChannel(Channel* a_channel)
    {
        int fd = a_channel->Fd();
        int ret = ::epoll_ctl(m_epfd_, EPOLL_CTL_DEL, fd, NULL);
        if (ret < 0)
        {
            //log
        }
    }

    void UpdateChannel(Channel* a_channel)
    {
        int fd = a_channel->Fd();
        struct epoll_event ee;
        memset(&ee, 0, sizeof(ee));
        ee.events = a_channel->Events();
        ee.data.ptr = a_channel;
        if (a_channel->GetInEpoll())
        {
            int ret = ::epoll_ctl(m_epfd_, EPOLL_CTL_MOD, fd, &ee);
            if (ret < 0)
            {
                //log
            }
        }
        else 
        {
            int ret = ::epoll_ctl(m_epfd_, EPOLL_CTL_ADD, fd, &ee);
            if (ret < 0)
            {
                //log
                return;
            }
            a_channel->SetInEpoll();
        }
    }
};
}


#endif