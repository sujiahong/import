#include "./toolbox/time_function.hpp"
#include "./toolbox/parallel.hpp"
#include "./multi_thread/thread_lock.hpp"
#include "./toolbox/simple_timer.hpp"
#include <unistd.h>
#include <unordered_set>
#include <iostream>

void TimerHandleFunc(unsigned long long a_timer_id)
{
    std::cout<<" 定时器id="<<a_timer_id<<std::endl;
}

int main(int argc, char** argv)
{
    su::SimpleTimer timer;
    timer.Start();
    //timer.RunAt(TimerHandleFunc, 1622210911);
    timer.RunEvery(TimerHandleFunc, 3, 2);
    while(1)
    {
        sleep(3);
    }
    return 0;
}