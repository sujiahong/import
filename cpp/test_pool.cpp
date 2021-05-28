#include "./multi_thread/thread_pool.hpp"
#include "./toolbox/parallel.hpp"
#include <unistd.h>
#include <iostream>


class CT: public su::TaskBase
{
public:
    void RunOnce(unsigned int a_thread_id)
    {
        std::cout<<" info: run once tid="<<pthread_self()<<" a_thread_id="<<a_thread_id<<std::endl;
    }
};

int main(int argc, char** argv)
{
    CT* ct = new CT();
    su::ThreadPool pool;
    pool.Start();
    pool.PushTask(ct);
    unsigned int n = pool.GetTaskNum();
    std::cout<<" info: GetTaskNum n="<<n<<std::endl;
    sleep(1);
    n = pool.GetTaskNum();
    std::cout<<" info: GetTaskNum n="<<n<<std::endl;
    sleep(2);
    // while (1)
    // {
    //     pool.PushTask(5);
    //     //sleep(1);
    // }
    return 0;
}