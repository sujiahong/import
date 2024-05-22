
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
#include <assert.h>

#include "../../toolbox/original_dependence.hpp"
#include "../../toolbox/time_function.hpp"
#include "epoll.h"
#include "channel.h"

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
    EventLoop():
        m_loop_(false),
        m_quit_(false),
        m_thread_id_(pthread_self())
    {
        m_active_channels_.clear();
        m_event_handling_ = false;
        m_cur_handle_channel_ = NULL;
        m_ep_ = new EPoll();
    }
    ~EventLoop();
    {
        delete m_ep_;
        m_cur_handle_channel_ = NULL;
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

// __thread EventLoop* t_loopInThisThread = NULL;

typedef struct ThreadParam
{
    EventLoopThreadPool* pool;
    unsigned int thd_index;
}THREAD_PARAM;

class EventLoopThreadPool: public Noncopyable
{
private:
    EventLoop* m_base_loop_;
    unsigned int m_thread_num_;
    bool m_running_;
    std::vector<EventLoop*> m_event_loops_;
    pthread_t m_tid_arr_[20];
    unsigned int m_next_;
public:
    EventLoopThreadPool(EventLoop* a_elp, unsigned int a_thd_num):
        m_base_loop_(a_elp),
        m_thread_num_(a_thd_num),
        m_running_(false),
        m_next_(0)
    {
    }
    ~EventLoopThreadPool()
    {
        // Don't delete loop, it's stack variable
    }
public:
    static void* ThreadFunc(void* a_arg)
    {
        THREAD_PARAM* param = (THREAD_PARAM*)a_arg;
        assert(!param);
        EventLoop* loop = new EventLoop();
        param->pool->m_event_loops_[param->thd_index] = loop;
        
        while (loop != 0)
        {
            loop->Loop();
        }
        return 0;
    }
    void Start()
    {
        m_event_loops_.resize(m_thread_num_);
        int ret = 0;
        for (unsigned int i = 0; i < m_thread_num_; ++i)
        {
            THREAD_PARAM param;
            param.pool = this;
            param.thd_index = i;
            ret = pthread_create(&(m_tid_arr_[i]), NULL, ThreadFunc, (void*)&param);
            if (ret != 0)
            {
                // std::cout<<" Error: create thread失败 ret="<<ret<<" i="<<i<<std::endl;
                continue;
            }
            // std::cout<<" Info: pthread_id="<<m_tid_arr_[i]<<std::endl;
            ret = pthread_detach(m_tid_arr_[i]);
            if (ret != 0)
            {
                // std::cout<<" Error: pthread_detach失败 ret="<<ret<<" i="<<i<<std::endl;
                continue;
            }
        }
        m_running_ = true;
    }
    void SetThreadNum(unsigned int a_thd_num)
    {
        m_thread_num_ = a_thd_num;
    }
    EventLoop* GetNextLoop()
    {
        unsigned int idx = m_next_ % m_thread_num_;
        ++m_next_;
        return m_event_loops_[idx];
    }

};

}//////namespace su


#endif