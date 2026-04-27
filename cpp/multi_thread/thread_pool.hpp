//////
////线程池
/////

#ifndef _THREAD_POOL_HPP_
#define _THREAD_POOL_HPP_

#include "../base/original_base.h"
#include "thread_lock.hpp"
#include "task_pool.hpp"
#include <pthread.h>
#include <list>
#include <atomic>

#define MAX_THREAD_NUM 50
#define DEFAULT_THREAD_NUM 10

namespace su
{
class ThreadPool: public Noncopyable
{
private:
    unsigned int m_thread_num_;
    std::atomic<bool> m_running_;
    pthread_t m_tid_arr_[MAX_THREAD_NUM];
    TaskPool<TaskBase*> m_task_pool_;
public:
    ThreadPool()
    {
        m_thread_num_ = DEFAULT_THREAD_NUM;
        Init();
    }
    ThreadPool(unsigned int a_num)
    {
        if (a_num > MAX_THREAD_NUM)
            a_num = MAX_THREAD_NUM;
        if (a_num < 1)
            a_num = 1;
        m_thread_num_ = a_num;
        Init();
    }
    ~ThreadPool()
    {
        Stop();
    }
private:
    static void* ThreadFunc(void* a_arg)
    {
        ThreadPool* self = static_cast<ThreadPool*>(a_arg);
        TaskBase* task_ptr = nullptr;
        while (self->m_running_)
        {
            self->PopTask(task_ptr);
            if (task_ptr == nullptr) break;
            task_ptr->RunOnce();
            if (task_ptr->ClearStat())
            {
                delete task_ptr;
                task_ptr = nullptr;
            }
        }
        return nullptr;
    }
    void Init()
    {
        m_running_ = true;
        int ret = 0;
        for (unsigned int i = 0; i < m_thread_num_; ++i)
        {
            ret = pthread_create(&(m_tid_arr_[i]), NULL, ThreadFunc, this);
            if (ret != 0)
            {
                std::cout<<" Error: create thread失败 ret="<<ret<<" i="<<i<<std::endl;
                continue;
            }
            ret = pthread_detach(m_tid_arr_[i]);
            if (ret != 0)
            {
                std::cout<<" Error: pthread_detach失败 ret="<<ret<<" i="<<i<<std::endl;
                continue;
            }
        }
    }
public:
    inline void Start()
    {
        m_running_ = true;
    }
    inline void Stop()
    {
        m_running_ = false;
        m_task_pool_.Stop();
    }
    inline void PopTask(TaskBase*& a_task_ptr)
    {
        m_task_pool_.PopTask(a_task_ptr);
    }
    inline void PushTask(TaskBase* a_task_ptr)
    {
        m_task_pool_.PushTask(a_task_ptr);
    }
    inline unsigned int GetTaskNum()
    {
        return m_task_pool_.TaskNum();
    }
    inline unsigned int GetThreadNum()
    {
        return m_thread_num_;
    }
};

}
#endif