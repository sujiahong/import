#include "./toolbox/time_function.hpp"
#include "./toolbox/parallel.hpp"
#include "./multi_thread/thread_lock.hpp"
#include <unistd.h>
//#include <sys/syscall.h>
#include <unordered_set>
#include <iostream>
#include <cstdlib>

std::unordered_set<unsigned long long> id_set;
su::LockMutex mutex;
unsigned int count = 0;

unsigned long long uuid()
{
    struct timespec ts;
    if (clock_gettime(CLOCK_REALTIME, &ts) != 0)
        return 0;
    pthread_t tid = pthread_self();
    unsigned long long td = tid;
    pid_t pid = getpid();
    unsigned long long pd = pid << 24;
    pd = (pd | td)<<32;
    srand((unsigned int)ts.tv_nsec);
    unsigned long long time = ts.tv_sec*1000000000+ts.tv_nsec;
    unsigned long long r = rand();
    r = (r<<30);
    //std::cout<<" r="<<r<<std::endl;
    time = (time|pd) + r;
    return time;
}

void* ThreadFunc(void* arg)
{
    // pthread_t tid = pthread_self();
    // std::cout<<" 线程id="<<tid<<std::endl;
    // pid_t pid = getpid();
    // std::cout<<" 进程id="<<pid<<std::endl;
    unsigned long long time;
    for (int i = 0; i < 10000000; ++i)
    {
        time = uuid();
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