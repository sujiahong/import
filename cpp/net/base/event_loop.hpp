
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
#include <mutex>
#include <pthread.h>
#include <assert.h>

#include <sys/eventfd.h>
#include <unistd.h>

#include "../../base/original_base.h"
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
    const pthread_t m_thread_id_;
    std::vector<Channel*> m_active_channels_;
    bool m_event_handling_;
    Channel* m_cur_handle_channel_;
    EPoll* m_ep_;          ////////////epoll

    /////跨线程任务队列 + eventfd 唤醒
    int m_wakeup_fd_;
    std::unique_ptr<Channel> m_wakeup_channel_;
    std::mutex m_mutex_;
    std::vector<std::function<void()>> m_pending_functors_;
    bool m_calling_pending_functors_;

public:
    EventLoop();
    ~EventLoop();

    void Loop();
    void Quit();
    bool IsInLoopThread();
    void AssertInLoopThread();

    void UpdateChannel(Channel* channel);
    void RemoveChannel(Channel* channel);
    bool hasChannel(Channel* channel);

    /// @brief 若在当前 loop 线程则直接执行，否则入队等待
    void RunInLoop(std::function<void()> cb);
    /// @brief 入队，并唤醒 loop 线程执行（用于跨线程投递任务）
    void QueueInLoop(std::function<void()> cb);
private:
    void Wakeup();
    void HandleWakeup();
    void DoPendingFunctors();
};

inline EventLoop::EventLoop():
    m_loop_(false),
    m_quit_(false),
    m_thread_id_(pthread_self()),
    m_event_handling_(false),
    m_cur_handle_channel_(NULL),
    m_calling_pending_functors_(false)
{
    m_active_channels_.clear();
    m_ep_ = new EPoll();

    ////eventfd 用于唤醒阻塞在 epoll_wait 上的 loop 线程
    m_wakeup_fd_ = ::eventfd(0, EFD_NONBLOCK | EFD_CLOEXEC);
    assert(m_wakeup_fd_ > 0);
    m_wakeup_channel_.reset(new Channel(this, m_wakeup_fd_));
    m_wakeup_channel_->SetReadCallback(std::bind(&EventLoop::HandleWakeup, this));
    m_wakeup_channel_->EnableRead();
}

inline EventLoop::~EventLoop()
{
    m_wakeup_channel_->DisableAll();
    m_wakeup_channel_.reset();
    ::close(m_wakeup_fd_);
    delete m_ep_;
    m_cur_handle_channel_ = NULL;
}

inline void EventLoop::Loop()
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

        ////处理跨线程投递的任务（此时已不在 Channel 回调栈中，可安全析构对象）
        DoPendingFunctors();
    }
}

inline void EventLoop::Quit()
{
    m_quit_ = true;
    if (!IsInLoopThread())
        Wakeup();
}

inline bool EventLoop::IsInLoopThread()
{
    return (m_thread_id_ == pthread_self());
}

inline void EventLoop::AssertInLoopThread()
{
    assert(IsInLoopThread());
}

inline void EventLoop::UpdateChannel(Channel* channel)
{
    m_ep_->UpdateChannel(channel);
}

inline void EventLoop::RemoveChannel(Channel* channel)
{
    if (m_event_handling_)
    {
        assert(m_cur_handle_channel_== channel 
            || std::find(m_active_channels_.begin(), m_active_channels_.end(), channel) == m_active_channels_.end());
    }
    m_ep_->RemoveChannel(channel);
}

inline bool EventLoop::hasChannel(Channel* channel)
{
    return m_ep_->hasChannel(channel);
}

inline void EventLoop::RunInLoop(std::function<void()> cb)
{
    if (IsInLoopThread())
    {
        cb();
    }
    else
    {
        QueueInLoop(std::move(cb));
    }
}

inline void EventLoop::QueueInLoop(std::function<void()> cb)
{
    {
        std::lock_guard<std::mutex> lock(m_mutex_);
        m_pending_functors_.push_back(std::move(cb));
    }
    ////不在 loop 线程，或正在执行 pending functors 时，需要唤醒
    if (!IsInLoopThread() || m_calling_pending_functors_)
        Wakeup();
}

inline void EventLoop::Wakeup()
{
    uint64_t one = 1;
    ssize_t n = ::write(m_wakeup_fd_, &one, sizeof(one));
    (void)n;
}

inline void EventLoop::HandleWakeup()
{
    uint64_t one = 1;
    ssize_t n = ::read(m_wakeup_fd_, &one, sizeof(one));
    (void)n;
    DoPendingFunctors();
}

inline void EventLoop::DoPendingFunctors()
{
    std::vector<std::function<void()>> functors;
    m_calling_pending_functors_ = true;
    {
        std::lock_guard<std::mutex> lock(m_mutex_);
        functors.swap(m_pending_functors_);
    }
    for (auto& f : functors)
    {
        f();
    }
    m_calling_pending_functors_ = false;
}

// __thread EventLoop* t_loopInThisThread = NULL;

class EventLoopThreadPool;

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
        assert(a_arg != NULL);
        std::unique_ptr<THREAD_PARAM> param(static_cast<THREAD_PARAM*>(a_arg));
        EventLoop* loop = new EventLoop();
        param->pool->m_event_loops_[param->thd_index] = loop;
        loop->Loop();
        return 0;
    }
    void Start()
    {
        m_event_loops_.resize(m_thread_num_);
        int ret = 0;
        for (unsigned int i = 0; i < m_thread_num_; ++i)
        {
            THREAD_PARAM* param = new THREAD_PARAM{this, i};
            ret = pthread_create(&(m_tid_arr_[i]), NULL, ThreadFunc, param);
            if (ret != 0)
            {
                delete param;
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
