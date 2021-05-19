#ifndef _PARALLEL_HPP_
#define _PARALLEL_HPP_

#include <functional>
#include <pthread.h>

namespace su
{
/////异步执行，保证主线程不会提前结束
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

}

#endif