//////linux线程锁
////
//// STL库保证多线程读，单线程读写。不保证多线程读写
////
////

#ifndef _THREAD_LOCK_HPP_
#define _THREAD_LOCK_HPP_

#include <pthread.h>
#include "../toolbox/original_dependence.hpp"
#include <iostream>

namespace su
{
////////////////////////递归锁/////////////////////////
class LockMutexRecursive: public Noncopyable
{
    pthread_mutex_t m_mutex_;
public:
    LockMutexRecursive()
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
        ret = pthread_mutex_init(&m_mutex_, &attr);
        if (ret != 0)
        {
            std::cout<<" Error: lock_init 锁初始化失败 ret="<<ret<<std::endl;
        }
    }
    ~LockMutexRecursive()
    {
        pthread_mutex_destroy(&m_mutex_);
    }
    inline void lock()
    {
        int ret = pthread_mutex_lock(&m_mutex_);
        if (ret != 0)
        {
            std::cout<<" Error: lock 加锁失败 ret="<<ret<<std::endl;
        }
    }
    inline void unlock()
    {
        int ret = pthread_mutex_unlock(&m_mutex_);
        if (ret != 0)
        {
            std::cout<<" Error: unlock 解锁失败 ret="<<ret<<std::endl;
        }
    }
    inline pthread_mutex_t* GetThreadMutex()// 仅供 Condition 调用，严禁自己调用
    {
        return &m_mutex_;
    }
};

/////////使用RAII控制加锁还是解锁
class LockMutexRecursiveGuard: public Noncopyable
{
    LockMutexRecursive& m_lock_mutex_;
public:
    LockMutexRecursiveGuard(LockMutexRecursive& a_mutex):m_lock_mutex_(a_mutex)
    {
        m_lock_mutex_.lock();
    }
    ~LockMutexRecursiveGuard()
    {
        m_lock_mutex_.unlock();
    }
};

////////////////////////非递归锁////////////////////////
class LockMutex: public Noncopyable
{
    pthread_mutex_t m_mutex_;
public:
    LockMutex()
    {
        int ret = pthread_mutex_init(&m_mutex_, 0);
        if (ret != 0)
        {
            std::cout<<" Error: lock_init 锁初始化失败 ret="<<ret<<std::endl;
        }
    }
    ~LockMutex()
    {
        pthread_mutex_destroy(&m_mutex_);
    }
    inline void lock()
    {
        int ret = pthread_mutex_lock(&m_mutex_);
        if (ret != 0)
        {
            std::cout<<" Error: lock 加锁失败 ret="<<ret<<std::endl;
        }
    }
    inline void unlock()
    {
        int ret = pthread_mutex_unlock(&m_mutex_);
        if (ret != 0)
        {
            std::cout<<" Error: unlock 解锁失败 ret="<<ret<<std::endl;
        }
    }
    inline pthread_mutex_t* GetThreadMutex()// 仅供 Condition 调用，严禁自己调用
    {
        return &m_mutex_;
    }
};

/////////使用RAII控制加锁还是解锁
class LockMutexGuard: public Noncopyable
{
    LockMutex& m_lock_mutex_;
public:
    LockMutexGuard(LockMutex& a_mutex):m_lock_mutex_(a_mutex)
    {
        m_lock_mutex_.lock();
    }
    ~LockMutexGuard()
    {
        m_lock_mutex_.unlock();
    }
};

////////////////////////条件变量/////////////////////////
class Condition: public Noncopyable
{
private:
    LockMutex& m_mutex_;
    pthread_cond_t m_cond_;

public:
    Condition(LockMutex& a_mutex):m_mutex_(a_mutex)
    {
        int ret = pthread_cond_init(&m_cond_, 0);
        if (ret != 0)
        {
            std::cout<<" Error: cond_init 条件变量初始化失败 ret="<<ret<<std::endl;
        }
    }

    ~Condition()
    {
        int ret = pthread_cond_destroy(&m_cond_);
        if (ret != 0)
        {
            std::cout<<" Error: cond_destory 条件变量销毁失败 ret="<<ret<<std::endl;
        }
    }
public:
/*
对于 wait() 端：

1. 必须与 mutex 一起使用，该布尔表达式的读写需受此 mutex 保护

2. 在 mutex 已上锁的时候才能调用 wait()

3. 把判断布尔条件和 wait() 放到 while 循环中
*/
    inline void Wait()
    {
        int ret = pthread_cond_wait(&m_cond_, m_mutex_.GetThreadMutex());
        if (ret != 0)
        {
            std::cout<<" Error: cond_wait 等待失败 ret="<<ret<<std::endl;
        }
    }
/*
对于 signal/broadcast 端：

1. 不一定要在 mutex 已上锁的情况下调用 signal （理论上）

2. 在 signal 之前一般要修改布尔表达式

3. 修改布尔表达式通常要用 mutex 保护（至少用作 full memory barrier）
*/
    inline void Notify()
    {
        int ret = pthread_cond_signal(&m_cond_);
        if (ret != 0)
        {
            std::cout<<" Error: cond_signal 通知失败 ret="<<ret<<std::endl;
        }
    }
    inline void NotifyAll()
    {
        int ret = pthread_cond_broadcast(&m_cond_);
        if (ret != 0)
        {
            std::cout<<" Error: cond_broadcast 广播失败 ret="<<ret<<std::endl;
        }
    }
};




}

/////////直接使用递归锁
#define MUTEX_RECURSIVE_GUARD(m) \
su::LockMutexRecursiveGuard g_recursive_guard(m);


/////////直接使用非递归锁
#define MUTEX_GUARD(m) \
su::LockMutexGuard g_guard(m);


#endif