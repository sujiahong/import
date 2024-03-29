
////////
/////
///////
#ifndef _EVENT_LOOP_HPP_
#define _EVENT_LOOP_HPP_

#include <atomic>
#include <unordered_map>
#include <vector>
#include <functional>
#include <memory>

#include <pthread.h>
#include <sys/epoll.h>
#include "../toolbox/original_dependence.hpp"
#include "../toolbox/time_function.hpp"
#include "tcp_connection.hpp"

#define MAX_EVENT_NUM 2000
const int kPollTimeMs = 10000;
namespace su
{

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
class EPoll: public Noncopyable
{
private:
    int m_ep_fd_;
    struct epoll_event m_event_arr_[MAX_EVENT_NUM];
public:
    EPoll(/* args */)
    {
        m_ep_fd_ = ::epoll_create1(EPOLL_CLOEXEC);
        assert(m_ep_fd > 0);
    }
    ~EPoll()
    {
    }
    const char* operationToString(int op)
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
    unsigned int Poll(int a_tmout)
    {

        int numEvents = ::epoll_wait(m_ep_fd_, m_event_arr_, MAX_EVENT_NUM, a_tmout);

        unsigned int now = (unsigned int)su::SecondTime();
        return now;
    }
    void Update(int a_operation, int a_fd, unsigned int a_evs)
    {
        struct epoll_event ee;
        memset(&ee, 0, sizeof(ee));
        ee.events = a_evs;
        ee.data.fd = a_fd;
        int ret = ::epoll_ctl(m_ep_fd_, a_operation, a_fd, &ee);
        if (ret < 0)
        {
            //log
            return;
        }
    }
};

class EventLoop: public Noncopyable
{
    typedef std::unordered_map<int, su::TcpConnectionPtr> CONNECTION_MAP_TYPE;
private:
    std::atomic<bool> m_loop_;
    std::atomic<bool> m_quit_;
    const unsigned int m_thread_id_;

    EPoll m_ep_;          ////////////epoll
    
    CONNECTION_MAP_TYPE m_connections_;//////所有连接
public:
    EventLoop():m_loop_(false),m_thread_id_(pthread_self()),m_quit_(false)
    {}
    ~EventLoop();
    {
    }
public:
    void Loop()
    {
        assert(!m_loop_)
        m_loop_ = true;
        m_quit_ = false;
        while (!m_quit_)
        {
            unsigned int recv_time = m_ep_.Poll(kPollTimeMs);
        }
        
    }
    void Quit()
    {
        m_quit_ = true;
    }
    bool IsInLoopThread()
    {
        return (m_thread_id_ == pthread_self());
    }
    void AssertInLoopThread()
    {} 
};

__thread EventLoop* t_loopInThisThread = NULL;


class EventLoopThreadPool: public Noncopyable
{
private:
    EventLoop* m_base_loop_;
    std::vector<EventLoop*> m_event_loops_;
    unsigned int m_thread_num_;
    pthread_t m_tid_arr_[20];
public:
    EventLoopThreadPool(EventLoop* a_elp, unsigned int a_thd_num)m_base_loop_(a_elp),m_thread_num_(a_thd_num)
    {
        int ret = 0;
        for (unsigned int i = 0; i < m_thread_num_; ++i)
        {
            ret = pthread_create(&(m_tid_arr_[i]), NULL, ThreadFunc, this);
            if (ret != 0)
            {
                std::cout<<" Error: create thread失败 ret="<<ret<<" i="<<i<<std::endl;
                continue;
            }
            std::cout<<" Info: pthread_id="<<m_tid_arr_[i]<<std::endl;
            ret = pthread_detach(m_tid_arr_[i]);
            if (ret != 0)
            {
                std::cout<<" Error: pthread_detach失败 ret="<<ret<<" i="<<i<<std::endl;
                continue;
            }
        }
    }
    ~EventLoopThreadPool()
    {}
public:
    EventLoop* GetNextLoop()
    {

    }

};

}


#endif