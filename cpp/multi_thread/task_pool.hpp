/////
///任务池 线程安全，任务基类
////

#ifndef _TASK_POOL_HPP_
#define _TASK_POOL_HPP_

#include "../toolbox/original_dependence.hpp"
#include "thread_lock.hpp"
#include <list>
#include <cassert>

namespace su
{

class TaskBase: public Noncopyable
{
public:
    TaskBase(){}
    virtual ~TaskBase(){}
    virtual void RunOnce(unsigned int a_thread_id=0){}//////////同步执行
    virtual void RunOnce(unsigned int a_thread_id, unsigned long long a_timer_id){}//////定时器使用，同步执行
    virtual bool ClearStat(){return true;}
};

template<typename T>
class TaskPool: public Noncopyable
{
private:
    mutable LockMutex m_mutex_;
    Condition m_cond_;   
    std::list<T> m_task_list_;
public:
    TaskPool():m_mutex_(),m_cond_(m_mutex_),m_task_list_()
    {}
    ~TaskPool()
    {
        MUTEX_GUARD(m_mutex_)
        m_cond_.NotifyAll();
    }
public:
    /////////////
    void PopTask(T& a_task)
    {
        MUTEX_GUARD(m_mutex_);
        while (m_task_list_.empty())
        {
            m_cond_.Wait();
        }
        assert(!m_task_list_.empty());
        a_task = std::move(m_task_list_.front());
        m_task_list_.pop_front();
        //std::cout<<" Info: 线程执行22222 tid="<<pthread_self()<<std::endl;
    }
    ///////////////
    void PushTask(const T& a_task)
    {
        MUTEX_GUARD(m_mutex_)
        m_task_list_.push_back(a_task);
        m_cond_.Notify();
    }
    ///////////////
    inline unsigned int TaskNum()
    {
        MUTEX_GUARD(m_mutex_)
        return m_task_list_.size();
    }
};

}

#endif