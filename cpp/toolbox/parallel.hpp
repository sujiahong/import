#ifndef _PARALLEL_HPP_
#define _PARALLEL_HPP_

#include <functional>
#include <pthread.h>
#include "../multi_thread/thread_pool.hpp"
#include <tuple>

/*
type __sync_fetch_and_add (type *ptr, type value, ...)
// 将value加到*ptr上，结果更新到*ptr，并返回操作之前*ptr的值
type __sync_fetch_and_sub (type *ptr, type value, ...)
// 从*ptr减去value，结果更新到*ptr，并返回操作之前*ptr的值
type __sync_fetch_and_or (type *ptr, type value, ...)
// 将*ptr与value相或，结果更新到*ptr， 并返回操作之前*ptr的值
type __sync_fetch_and_and (type *ptr, type value, ...)
// 将*ptr与value相与，结果更新到*ptr，并返回操作之前*ptr的值
type __sync_fetch_and_xor (type *ptr, type value, ...)
// 将*ptr与value异或，结果更新到*ptr，并返回操作之前*ptr的值
type __sync_fetch_and_nand (type *ptr, type value, ...)
// 将*ptr取反后，与value相与，结果更新到*ptr，并返回操作之前*ptr的值
type __sync_add_and_fetch (type *ptr, type value, ...)
// 将value加到*ptr上，结果更新到*ptr，并返回操作之后新*ptr的值
type __sync_sub_and_fetch (type *ptr, type value, ...)
// 从*ptr减去value，结果更新到*ptr，并返回操作之后新*ptr的值
type __sync_or_and_fetch (type *ptr, type value, ...)
// 将*ptr与value相或， 结果更新到*ptr，并返回操作之后新*ptr的值
type __sync_and_and_fetch (type *ptr, type value, ...)
// 将*ptr与value相与，结果更新到*ptr，并返回操作之后新*ptr的值
type __sync_xor_and_fetch (type *ptr, type value, ...)
// 将*ptr与value异或，结果更新到*ptr，并返回操作之后新*ptr的值
type __sync_nand_and_fetch (type *ptr, type value, ...)
// 将*ptr取反后，与value相与，结果更新到*ptr，并返回操作之后新*ptr的值
bool __sync_bool_compare_and_swap (type *ptr, type oldval type newval, ...)
// 比较*ptr与oldval的值，如果两者相等，则将newval更新到*ptr并返回true
type __sync_val_compare_and_swap (type *ptr, type oldval type newval, ...)
// 比较*ptr与oldval的值，如果两者相等，则将newval更新到*ptr并返回操作之前*ptr的值
__sync_synchronize (...)
// 发出完整内存栅栏
type __sync_lock_test_and_set (type *ptr, type value, ...)
// 将value写入*ptr，对*ptr加锁，并返回操作之前*ptr的值。即，try spinlock语义
void __sync_lock_release (type *ptr, ...)
// 将0写入到*ptr，并对*ptr解锁。即，unlock spinlock语义
*/


namespace su
{
/////异步执行，保证主线程不会提前结束 ////创建线程
static int AsyncExecute(void* (*a_func)(void*) , void* a_arg)
{
    int ret = 0;
    if (a_func == NULL)
        return -1;
    pthread_t tid;
    ret = pthread_create(&tid, NULL, a_func, a_arg);
    if (ret != 0)
    {
        return ret;
    }
    ret = pthread_detach(tid);
    if (ret != 0)
    {
        return ret;
    }
    return ret;
}


/////同步执行
static int SyncExecute(void* (*a_func)(void*) , void* a_arg)
{
    int ret = 0;
    if (a_func == NULL)
        return -1;
    pthread_t tid;
    ret = pthread_create(&tid, NULL, a_func, a_arg);
    if (ret != 0)
    {
        return ret;
    }
    ret = pthread_join(tid, NULL);
    if (ret != 0)
    {
        return ret;
    }
    return ret;
}

////////////////////////////////////////////////////////////////////
template<typename... Args>
class CommonTask: public su::TaskBase
{
private:
    std::tuple<Args...> args_;
    std::function<void(const std::tuple<Args...>&)> func_;
public:
    CommonTask(std::function<void(const std::tuple<Args...>&)> func, Args... args):func_(func),args_(args...)
    {}
    ~CommonTask()
    {}
    void RunOnce(unsigned int a_thread_id)
    {
        func_(args_);
    }
};
class CommonTaskWithoutParm: public su::TaskBase
{
private:
    std::function<void()> func_;
public:
    CommonTaskWithoutParm(std::function<void()> func):func_(func)
    {}
    ~CommonTaskWithoutParm()
    {}
    virtual void RunOnce(unsigned int a_thread_id)
    {
        func_();
    }
};

class CommonMultiWorks
{
private:
    su::ThreadPool td_pool_;
public:
    CommonMultiWorks(unsigned int a_num):td_pool_(a_num)
    {
        td_pool_.Start();
    }
    ~CommonMultiWorks()
    {}

    // template<typename... Args>
    //void RunTask(std::function<void(const std::tuple<Args...>&)> func, Args... args);
    inline void RunTask(std::function<void()> func)
    {
        td_pool_.PushTask(new CommonTaskWithoutParm(func));
    }
};
static CommonMultiWorks* g_multi_works = new CommonMultiWorks(4);
/////////线程池异步执行
static void AsyncExecute(std::function<void()> a_func)//////对外接口
{
    g_multi_works->RunTask(a_func);
}

}

#endif