//////linux线程锁

#ifndef _THREAD_LOCK_HPP_
#define _THREAD_LOCK_HPP_

#include <pthread.h>
#include <iostream>

namespace su
{

class LockMutex
{
    pthread_mutex_t m_mutex;
public:
    LockMutex()
    {
        pthread_mutexattr_t attr;
        int ret = pthread_mutexattr_init(&attr);
        if (ret != 0)
        {
            std::cout<<" Error: attr_init 锁属性初始化失败 ret="<<ret<<std::endl;
        }
        ret = pthread_mutexattr_settype(&attr, PTHREAD_MUTEX_RECURSIVE);
        if (ret != 0)
        {
            std::cout<<" Error: attr_settype 锁属性设置失败 ret="<<ret<<std::endl;
        }
        ret = pthread_mutex_init(&m_mutex, &attr);
        if (ret != 0)
        {
            std::cout<<" Error: lock_init 锁初始化失败 ret="<<ret<<std::endl;
        }
    }
    ~LockMutex()
    {
        pthread_mutex_destroy(&m_mutex);
    }
    inline void lock()
    {
        int ret = pthread_mutex_lock(&m_mutex);
        if (ret != 0)
        {
            std::cout<<" Error: lock 加锁失败 ret="<<ret<<std::endl;
        }
    }
    inline void unlock()
    {
        int ret = pthread_mutex_unlock(&m_mutex);
        if (ret != 0)
        {
            std::cout<<" Error: unlock 解锁失败 ret="<<ret<<std::endl;
        }
    }
};

/////////使用RAII控制加锁还是解锁
class LockMutexGuard
{
    LockMutex& m_lock_mutex;
public:
    LockMutexGuard(LockMutex& a_mutex):m_lock_mutex(a_mutex)
    {
        m_lock_mutex.lock();
    }
    ~LockMutexGuard()
    {
        m_lock_mutex.unlock();
    }
};


}

/////////直接使用
#define MUTEX_GUARD(m) \
su::LockMutexGuard guard(m);


#endif