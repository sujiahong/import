#include "./toolbox/time_function.hpp"
#include "./toolbox/parallel.hpp"
#include "./multi_thread/thread_lock.hpp"
#include "./toolbox/simple_timer.hpp"
#include "./multi_thread/task_pool.hpp"
#include <unistd.h>
#include <unordered_set>
#include <iostream>

typedef void(*TimerFunc)(unsigned long long);

void TimerHandleFunc(unsigned long long a_timer_id)
{
    std::cout<<" 定时器id="<<a_timer_id<<std::endl;
}

class TestTask: public su::TaskBase
{
private:
    TimerFunc m_func_ptr;
public:
    TestTask(TimerFunc a_ptr):m_func_ptr(a_ptr)
    {}
    ~TestTask()
    {}
    void RunOnce(unsigned int a_thread_id, unsigned long long a_timer_id)
    {
        std::cout<<" 线程id="<<a_thread_id<<std::endl;
        m_func_ptr(a_timer_id);
    }
};

int main(int argc, char** argv)
{
    su::SimpleTimer timer;
    timer.Start();
    //timer.RunAt(new TestTask(TimerHandleFunc), 1622269179);
    timer.RunEvery(new TestTask(TimerHandleFunc), 2, 2);
    while(1)
    {
        sleep(3);
    }
    return 0;
}