
////////
///// 事件循环
////////
#ifndef _EVENT_LOOP_HPP_
#define _EVENT_LOOP_HPP_

#include <atomic>
#include <unordered_map>
#include <vector>
#include <functional>
#include <memory>

#include <pthread.h>
#include "../../toolbox/original_dependence.hpp"
#include "../../toolbox/time_function.hpp"
#include "tcp_connection.hpp"


const int kPollTimeMs = 10000;
namespace su
{

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
    EventLoopThreadPool(EventLoop* a_elp, unsigned int a_thd_num):m_base_loop_(a_elp),m_thread_num_(a_thd_num)
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