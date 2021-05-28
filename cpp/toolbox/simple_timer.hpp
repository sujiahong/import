///
///简单定时器
///

#ifndef _TIMER_HPP_
#define _TIMER_HPP_

#include <set>
#include <vector>
#include <pthread.h>
#include <unordered_map>
#include <unistd.h>
#include <climits>
#include "../multi_thread/thread_lock.hpp"
#include "../multi_thread/thread_pool.hpp"
#include "../multi_thread/task_pool.hpp"
#include "time_function.hpp"
#include "original_dependence.hpp"
//#include <bl_const/Debug_log.h>

namespace su
{

typedef void (*FuncPtr)(unsigned long long);

struct TimerKey
{
    unsigned long long expire_time;
    unsigned long long timer_id;
    TimerKey()
    {
        expire_time = 0;
        timer_id = 0;
    }
    bool operator < (const TimerKey& a_other) const
    {
        if(expire_time == a_other.expire_time)
            return (timer_id < a_other.timer_id);
        return (expire_time < a_other.expire_time);
    }
};

class TaskTimer: public TaskBase
{
private:
    unsigned long long m_timer_id_;
    FuncPtr m_func_ptr_;
public:
    TaskTimer(unsigned long long a_timer_id, FuncPtr a_func_ptr):m_timer_id_(a_timer_id),m_func_ptr_(a_func_ptr)
    {}
    ~TaskTimer()
    {}
    void RunOnce(unsigned int a_thread_id)//////ron
    {
        if (m_func_ptr_)
            m_func_ptr_(m_timer_id_);
    }
};

struct TimerValue
{
    unsigned int interval;
    int count;
    FuncPtr handle_func;
    TimerValue()
    {
        interval = 0;
        count = 0;
        handle_func = 0;
    }
};

class SimpleTimer: public Noncopyable
{
private:
    std::set<TimerKey> m_timer_set_;
    std::unordered_map<unsigned long long, TimerValue> m_timer_handler_map_;
    struct TimerKey m_tmp_key_;/////////插入，查找临时使用
    struct TimerValue m_tmp_val_;/////////插入，查找临时使用
    mutable LockMutex m_timer_mutex_;
    bool m_running_;
    ThreadPool m_pool_;
private:
    void FindExpiredAndHandle()
    {
        MUTEX_GUARD(m_timer_mutex_)
        m_tmp_key_.expire_time = second_time();
        m_tmp_key_.timer_id = ULLONG_MAX;
        std::set<struct TimerKey>::iterator itor = m_timer_set_.lower_bound(m_tmp_key_);
        std::vector<struct TimerKey> vec(m_timer_set_.begin(), itor);
        std::unordered_map<unsigned long long, TimerValue>::iterator map_itor = m_timer_handler_map_.end();
        for (auto it = vec.begin(); it != vec.end();++it)
        {
            //LOG_TRACE(3, true, "FindExpiredAndHandle ", __LINE__<<" Info: 到期时间 expire_time="<<it->expire_time<<" timer_id="<<it->timer_id<<" now="<<m_tmp_item_.expire_time);
            map_itor = m_timer_handler_map_.find(it->timer_id);
            if (map_itor != m_timer_handler_map_.end())
            {
                if (map_itor->second.count< 0 && map_itor->second.interval != 0)
                {
                    m_tmp_key_.expire_time = it->expire_time + map_itor->second.interval;
                    m_tmp_key_.timer_id = it->timer_id;
                    m_timer_set_.insert(m_tmp_key_);
                    m_timer_set_.erase(*it);
                    m_pool_.PushTask(new TaskTimer(it->timer_id, map_itor->second.handle_func));
                    continue;
                }
                if (map_itor->second.count < 1)
                {
                    m_pool_.PushTask(new TaskTimer(it->timer_id, map_itor->second.handle_func));
                    m_timer_handler_map_.erase(map_itor);
                    m_timer_set_.erase(*it);
                }
                else
                {
                    --(map_itor->second.count);
                    m_tmp_key_.expire_time = it->expire_time + map_itor->second.interval;
                    m_tmp_key_.timer_id = it->timer_id;
                    m_timer_set_.insert(m_tmp_key_);
                    m_timer_set_.erase(*it);
                    m_pool_.PushTask(new TaskTimer(it->timer_id, map_itor->second.handle_func));
                }
            }
            else////////没有处理，删除定时器
            {
                m_timer_set_.erase(*it);
            }
        }
    }
    static void* ThreadFunc(void* arg)
    {
        SimpleTimer* self = (SimpleTimer*)arg;
        while(1)
        {
            if(self->m_running_)
            {
                self->FindExpiredAndHandle();
                //self->HandleExpired(vec);
            }
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
        m_pool_.Start();
        m_running_ = true;
    }

    inline unsigned long long RunAt(FuncPtr a_ptr, unsigned long long a_timestamp)
    {
        MUTEX_GUARD(m_timer_mutex_)
        m_tmp_key_.timer_id = nano_time();
        m_tmp_key_.expire_time = a_timestamp;
        m_timer_set_.insert(m_tmp_key_);
        m_tmp_val_.interval = 0;
        m_tmp_val_.count = 0;
        m_tmp_val_.handle_func = a_ptr;
        m_timer_handler_map_[m_tmp_key_.timer_id] = m_tmp_val_;
        return  m_tmp_key_.timer_id;
    }

    inline unsigned long long RunAfter(FuncPtr a_ptr, unsigned long long a_when)
    {
        return RunAt(a_ptr, second_time()+a_when);
    }

    inline unsigned long long RunEvery(FuncPtr a_ptr, unsigned long long a_interval, int a_count = -1)
    {
        MUTEX_GUARD(m_timer_mutex_)
        m_tmp_key_.timer_id = nano_time();
        m_tmp_key_.expire_time = second_time()+a_interval;
        m_timer_set_.insert(m_tmp_key_);
        m_tmp_val_.interval = (unsigned int)a_interval;
        m_tmp_val_.count = a_count;
        m_tmp_val_.handle_func = a_ptr;
        m_timer_handler_map_[m_tmp_key_.timer_id] = m_tmp_val_;
        return  m_tmp_key_.timer_id;
    }

    inline void SetRepeatCount(unsigned long long a_itmer_id, int a_count)//////设置重复次数 -1 无限循环
    {
        MUTEX_GUARD(m_timer_mutex_)
        std::unordered_map<unsigned long long, TimerValue>::iterator itor = m_timer_handler_map_.find(a_itmer_id);
        if (itor != m_timer_handler_map_.end())
        {
            itor->second.count = a_count;
        }
    }

    inline void Cancel(unsigned long long a_id)
    {
        MUTEX_GUARD(m_timer_mutex_)
        std::unordered_map<unsigned long long, TimerValue>::iterator map_itor = m_timer_handler_map_.find(a_id);
        if (map_itor != m_timer_handler_map_.end())
            m_timer_handler_map_.erase(map_itor);
    }

};



}


#endif