
//#include "./toolbox/parallel.hpp"
#include <set>
#include <vector>
#include <iostream>
#include <unistd.h>
#include <string.h>
#include "./algorithm/random_disorder.hpp"
#include "./toolbox/time_function.hpp"
#include "./toolbox/file_function.hpp"
#include <sys/mman.h>

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
    srand(su::MilliTime());
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
    //std::cout <<" count="<<count<<std::endl;
    char* maddr = su::FileOpenWithMMap("t2.txt", O_RDWR|O_CREAT|O_APPEND, 0766);
    std::cout <<" maddr="<<(void*)maddr<<std::endl;
    if (maddr)
    {
        memcpy(maddr, "dhhfkhjdlfggfddgdfsfs\n", 10); 
    }
    msync(maddr, 4096, MS_SYNC);
    munmap(maddr, 4096);
    return 0;
}