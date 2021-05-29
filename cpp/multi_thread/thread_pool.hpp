//////
////线程池,,,创建线程池的程序不能提前结束
/////

#ifndef _THREAD_POOL_HPP_
#define _THREAD_POOL_HPP_

#include "../toolbox/original_dependence.hpp"
#include "thread_lock.hpp"
#include "task_pool.hpp"
#include <pthread.h>
#include <list>

#define MAX_THREAD_NUM 50
#define DEFAULT_THREAD_NUM 10

namespace su
{
class ThreadPool: public Noncopyable
{
private:
    unsigned int m_thread_num_;
    pthread_t m_tid_arr_[MAX_THREAD_NUM];
    TaskPool<TaskBase*> m_task_pool_;
    bool m_running_;
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
        m_thread_num_ = a_num;
        Init();
    }
    ~ThreadPool()
    {
        m_running_= false;
    }
private:
    static void* ThreadFunc(void* a_arg)
    {
        ThreadPool* self = (ThreadPool*)a_arg;
        TaskBase* task_ptr = 0;
        while (self != 0)
        {
            if (self->m_running_)
            {
                self->PopTask(task_ptr);
                task_ptr->RunOnce();
                if (task_ptr->ClearStat())
                {
                    std::cout<<" Info: 删除内存 task_ptr="<<task_ptr<<std::endl;
                    delete task_ptr;
                    task_ptr = 0;
                }
            }
        }
        return 0;
    }
    void Init()
    {
        m_running_ = false;
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
public:
    inline void Start()
    {
        m_running_ = true;
    }
    /////////////
    inline void PopTask(TaskBase*& a_task_ptr)
    {
        m_task_pool_.PopTask(a_task_ptr);
    }
    ///////////////
    inline void PushTask(TaskBase* a_task_ptr)
    {
        m_task_pool_.PushTask(a_task_ptr);
    }
    ////////////
    inline unsigned int GetTaskNum()
    {
        return m_task_pool_.TaskNum();
    }
    ////////////
    inline unsigned int GetThreadNum()
    {
        return m_thread_num_;
    }
};


}
#endif