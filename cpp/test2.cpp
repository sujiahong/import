
//#include "./toolbox/parallel.hpp"
#include <set>
#include <vector>
#include <iostream>
#include <unistd.h>
#include "./algorithm/random_disorder.hpp"
#include "./toolbox/time_function.hpp"


int num = 0;

void* thread_func(void* arg)
{
    for (int i = 0; i < 10000; ++i)
    {
        num += 1;
    }
    return 0;
}

int main(int argc, char** argv)
{
    // su::AsyncExecute(thread_func, NULL);
    // sleep(2);
    // std::cout <<" num="<<num<<std::endl;

    // su::SyncExecute(thread_func, NULL);
    // std::cout <<" num="<<num<<std::endl;
    srand(su::milli_time());
    int r = 0;
    int count = 0;
    for (int i = 0; i < 10; ++i)
    {
        r = su::RangeRandom(1, 10);
        if (r > 5)
        {
            count++;
        }
    }
    std::cout <<" count="<<count<<std::endl;
    return 0;
}