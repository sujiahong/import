#include "./toolbox/time_function.hpp"
#include "./toolbox/parallel.hpp"
#include "./multi_thread/thread_lock.hpp"
#include <unistd.h>
#include <sys/syscall.h>

#include <unordered_set>
#include <iostream>
#include <cstdlib>

std::unordered_set<unsigned long long> id_set;
su::LockMutex mutex;
unsigned int count = 0;
static unsigned long long g_last_time = 0;
su::LockMutex last_time_mutex;
pid_t gettid()
{
    return syscall(SYS_gettid);
}

unsigned long long uuid(pid_t a_pid, pid_t a_tid)
{
    unsigned int count = 0;
    struct timespec ts;
    if (clock_gettime(CLOCK_REALTIME, &ts) != 0)
        return 0;
    // unsigned long long pd = a_pid<<16;
    // unsigned long long td = a_tid;
    // pd = (pd | td)<<32;
    unsigned long long r = 0;
    unsigned long long time = ts.tv_sec*1000000000+ts.tv_nsec;
    MUTEX_GUARD(last_time_mutex);
    if (time > g_last_time)
    {
        //std::cout<<" time="<<time<<" last_time="<<last_time<<std::endl;
        g_last_time = time;
    }
    // else if (time == g_last_time)
    // {
    //     // srand((unsigned int)ts.tv_nsec);
    //     // r = rand();
    //     //std::cout<<" time="<<time<<std::endl;
    //     ++g_last_time;
    // }
    else
    {
        ++g_last_time;
    }
    
    //r = (r<<30);
    //std::cout<<" r="<<r<<std::endl;
    //time = time + r;
    // td = ts.tv_nsec;
    // pd = (pd | td) + r;
    return g_last_time;
}

void* ThreadFunc(void* arg)
{
    // pthread_t tid = pthread_self();
    // std::cout<<" 线程id="<<tid<<std::endl;
    pid_t pid = getpid();
    std::cout<<" 进程id="<<pid<<std::endl;
    pid_t tid = gettid();
    std::cout<<" 线程id="<<tid<<std::endl;
    unsigned long long time;
    for (int i = 0; i < 10000000; ++i)
    {
        time = uuid(pid, tid);
        {
            MUTEX_GUARD(mutex);
            count++;
            if (id_set.find(time) != id_set.end())
            {
                std::cout<<" 已存在这个id="<<time<<" count="<<count<<std::endl;
            }
            else
            {
                id_set.insert(time);
            }
        }
    }

}

int main(int argc, char** argv)
{
    for (int i = 0; i < 20; ++i)
    {
        su::AsyncExecute(ThreadFunc, 0);
    }
    ThreadFunc(0);
    sleep(5);
    //std::cout<<" uuid="<<uuid()<<std::endl;
    return 0;
}