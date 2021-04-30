
#include "./toolbox/simple_cache.hpp"
#include <set>
#include <vector>
#include <iostream>
#include <pthread.h>
#include "./multi_thread/thread_lock.hpp"

static int num = 0;
su::LockMutex g_mutex;

void* thread_func(void* arg)
{
    //MUTEX_GUARD(g_mutex);
    for (int i = 0; i < 10000; ++i)
    {
        num += 1;
    }
    return 0;
}

int main(int argc, char** argv)
{

    su::SimpleCacheMap<int, int, 500> cache;
    cache.SetCache(1, 2);
    cache.SetCache(2, 648475);

    std::cout << " size="<< cache.GetCacheSize()<<" ttl="<<cache.GetTTL()<<std::endl;
    int val;
    int ret = cache.GetCache(2, val);
    if (ret != 0)
    {
        std::cout << " ret="<<ret<<std::endl;
        return ret;
    }
    std::cout <<" val="<<val<<std::endl;
    ////////////////////////////////////////////////////////
    pthread_t tid;
    pthread_create(&tid, NULL, thread_func, NULL);
    {
        //MUTEX_GUARD(g_mutex);
        for (int i = 0; i < 10000; ++i)
        {
            num += 1;
        }
    }
    pthread_join(tid, NULL);
    std::cout <<" num="<<num<<std::endl;
    return 0;
}