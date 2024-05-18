
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
#include "epoll.h"

const int kPollTimeMs = 10000;
namespace su
{

class EventLoop: public Noncopyable
{
private:
    std::atomic<bool> m_loop_;
    std::atomic<bool> m_quit_;
    const unsigned int m_thread_id_;
    std::vector<Channel*> m_active_channels_;
    bool m_event_handling_;
    Channel* m_cur_handle_channel_;
    EPoll* m_ep_;          ////////////epoll
    
public:
    EventLoop():m_loop_(false),m_quit_(false),m_thread_id_(pthread_self())
    {
        m_active_channels_.clear();
        m_event_handling_ = false;
        m_cur_handle_channel_ = NULL;
        m_ep_ = new EPoll();
    }
    ~EventLoop();
    {
    }
public:
    void Loop()
    {
        assert(!m_loop_);
        m_loop_ = true;
        m_quit_ = false;
        while (!m_quit_)
        {
            m_active_channels_.clear();
            unsigned int rt_time = m_ep_->Poll(kPollTimeMs, m_active_channels_);
            m_event_handling_ = true;
            for (Channel* channel : m_active_channels_)
            {
                m_cur_handle_channel_ = channel;
                m_cur_handle_channel_->HandleEvent(rt_time);
            }
            m_cur_handle_channel_ = NULL;
            m_event_handling_ = false;
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

    void UpdateChannel(Channel* channel)
    {
        m_ep_->UpdateChannel(channel);
    }
    void RemoveChannel(Channel* channel)
    {
        if (m_event_handling_)
        {
            assert(m_cur_handle_channel_== channel 
                || std::find(m_active_channels_.begin(), m_active_channels_.end(), channel) == m_active_channels_.end());
        }
        m_ep_->RemoveChannel(channel);
    }
    bool hasChannel(Channel* channel)
    {

    }
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

}//////namespace su


#endif