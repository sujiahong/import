
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
#include "./data_structure/bytes.h"
#include "./algorithm/hash.h"

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
        std::cout <<" r="<<r<<std::endl;
        if (r > 5)
        {
            count++;
        }
    }
    std::cout <<" count="<<count<<std::endl;
    char* maddr = su::FileOpenWithMMap("t2.txt", O_RDWR|O_CREAT|O_APPEND, 0766);
    std::cout <<" maddr="<<(void*)maddr<<std::endl;
    if (maddr)
    {
        memcpy(maddr, "dhhfkhjdlfggfddgdfsfs\n", 10); 
    }
    msync(maddr, 4096, MS_SYNC);
    munmap(maddr, 4096);
    unsigned int dateYM = su::DateYearMonth();
    unsigned int dateYMD = su::DateYearMonthDay();
    std::cout <<" dateYM="<<dateYM<<" dateYMD="<<dateYMD<<std::endl;
    std::string dateYMStr = su::DateYearMonthString();
    std::string dateYMDStr = su::DateYearMonthDayString();
    std::cout <<" dateYMStr="<<dateYMStr<<" dateYMDStr="<<dateYMDStr<<std::endl;
    std::string str("320384");
    su::Bytes b1(20), b2("3433", 4), b3(str), b4("3934wj");
    b1 = b2;
    std::cout <<" b1="<<b1.GetData()<<" b1len="<<b1.GetLen()
            <<" b2="<<b2.GetData()<<" b2len="<<b2.GetLen()
            <<" b3="<<b3.GetData()<<" b3len="<<b3.GetLen()
            <<" b4="<<b4.GetData()<<" b4len="<<b4.GetLen()
            <<" b3==b4 "<<(b3 == b3)
            << std::endl;

    uint32_t h1 = su::djb2_hash("394");
    uint32_t h2 = su::fnv1a_hash("394");
    uint64_t h3 = su::supcHash("394", 3);
    std::cout <<" h1="<<h1<<" h2="<<h2<<" h3="<<h3<<std::endl;
    return 0;
}