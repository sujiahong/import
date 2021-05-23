///
///简单定时器
///

#ifndef _TIMER_HPP_
#define _TIMER_HPP_

#include <set>
#include <pthread.h>
#include <unordered_map>
#include <unistd.h>
#include <climits>
#include "../multi_thread/thread_lock.hpp"
#include "time_function.hpp"
#include "original_dependence.hpp"
#include <bl_const/Debug_log.h>

namespace su
{

typedef void (*FuncPtr)(unsigned long long);

struct TimerItem
{
    unsigned long long expire_time;
    unsigned long long timer_id;
    unsigned int interval;
    int count;
    TimerItem()
    {
        expire_time = 0;
        timer_id = 0;
        interval = 0;
        count = 0;
    }
    bool operator < (const TimerItem& a_other) const
    {
        if(expire_time == a_other.expire_time)
            return (timer_id < a_other.timer_id);
        return (expire_time < a_other.expire_time);
    }
};

class SimpleTimer: public Noncopyable
{
private:
    std::set<struct TimerItem> m_timer_set_;
    std::unordered_map<unsigned long long, FuncPtr> m_timer_handler_map_;
    struct TimerItem m_tmp_item_;/////////插入，查找临时使用
    mutable LockMutex m_timer_mutex_;
    bool m_running_;
private:
    void FindExpiredAndHandle()
    {
        MUTEX_GUARD(m_timer_mutex_)
        m_tmp_item_.expire_time = second_time();
        m_tmp_item_.timer_id = ULLONG_MAX;
        std::set<struct TimerItem>::iterator itor = m_timer_set_.lower_bound(m_tmp_item_);
        std::unordered_map<unsigned long long, FuncPtr>::iterator map_itor = m_timer_handler_map_.end();
        for (auto it = m_timer_set_.begin(); it != itor;)
        {
            //LOG_TRACE(3, true, "FindExpiredAndHandle ", __LINE__<<" Info: 到期时间 expire_time="<<it->expire_time<<" timer_id="<<it->timer_id<<" now="<<m_tmp_item_.expire_time);
            map_itor = m_timer_handler_map_.find(it->timer_id);
            if (map_itor != m_timer_handler_map_.end())
            {
                if (map_itor->second)
                {
                    (map_itor->second)(it->timer_id);
                }
                if (it->count < 0 && it->interval != 0)
                {
                    it->expire_time += it->interval;
                    ++it;
                    continue;
                }
                (it->count)--;
                if (it->count < 1)
                {
                    m_timer_handler_map_.erase(map_itor);
                    m_timer_set_.erase(it++);
                }
                else
                {
                    it->expire_time += it->interval;
                    ++it;
                }
            }
            else
            {
                (it->count)--;
                if (it->count < 1)
                    m_timer_set_.erase(it++);
                else
                {
                    it->expire_time += it->interval;
                    ++it;
                }
            }
        }
        //m_timer_set_.erase(m_timer_set_.begin(), itor);
    }

    static void* ThreadFunc(void* arg)
    {
        SimpleTimer* self = (SimpleTimer*)arg;
        while(1)
        {
            if(self->m_running_)
            {
                self->FindExpiredAndHandle();
            }
            else
                usleep(10000);
        }
        return 0;
    }

public:
    SimpleTimer()
    {
        int ret = 0;
        pthread_t tid = 0;
        ret = pthread_create(&tid, NULL, ThreadFunc, this);
        if (ret != 0)
        {
            return;
        }
        ret = pthread_detach(tid);
        if (ret != 0)
        {
            return;
        }
        m_running_ = false;
    }

    void InitTimerDataFromRedis()
    {}

    ~SimpleTimer()
    {
        m_running_ = false;
    }

public:
    inline void Start()
    {
        m_running_ = true;
    }

    inline unsigned long long RunAt(FuncPtr a_ptr, unsigned long long a_timestamp)
    {
        MUTEX_GUARD(m_timer_mutex_)
        m_tmp_item_.timer_id = nano_time();
        m_tmp_item_.expire_time = a_timestamp;
        m_tmp_item_.interval = 0;
        m_tmp_item_.count = 0;
        m_timer_set_.insert(m_tmp_item_);
        m_timer_handler_map_[m_tmp_item_.timer_id] = a_ptr;
    }

    inline unsigned long long RunAfter(FuncPtr a_ptr, unsigned long long a_when)
    {
        RunAt(a_ptr, second_time()+a_when);
    }

    inline unsigned long long RunEvery(FuncPtr a_ptr, unsigned long long a_interval, int a_count = -1)
    {
        MUTEX_GUARD(m_timer_mutex_)
        m_tmp_item_.timer_id = nano_time();
        m_tmp_item_.expire_time = second_time()+a_interval;
        m_tmp_item_.interval = (unsigned int)a_interval;
        m_tmp_item_.count = a_count;
        m_timer_set_.insert(m_tmp_item_);
        m_timer_handler_map_[m_tmp_item_.timer_id] = a_ptr;
    }

    inline void SetRepeatCount(unsigned long long a_itmer_id, int a_count)//////设置重复次数 -1 无限循环
    {
        MUTEX_GUARD(m_timer_mutex_)
        std::set<struct TimerItem>::iterator itor = m_timer_set_.find(a_itmer_id);
        if (itor != m_timer_set_.end())
        {
            itor->count = a_count;
        }
    }

    inline void Cancel(unsigned long long a_id)
    {
        MUTEX_GUARD(m_timer_mutex_)
        std::unordered_map<unsigned long long, FuncPtr>::iterator map_itor = m_timer_handler_map_.find(a_id);
        if (map_itor != m_timer_handler_map_.end())
            m_timer_handler_map_.erase(map_itor);
    }

};



}


#endif