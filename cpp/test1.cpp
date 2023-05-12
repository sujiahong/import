
#include "./toolbox/simple_cache.hpp"
#include <set>
#include <vector>
#include <iostream>
#include <pthread.h>
#include "./multi_thread/thread_lock.hpp"
#include "spdlog/spdlog.h"
#include "spdlog/sinks/basic_file_sink.h"
#include "spdlog/sinks/rotating_file_sink.h"
#include "spdlog/sinks/stdout_color_sinks.h"
#include "spdlog/async.h"

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
    spdlog::init_thread_pool(8192, 4);
    auto logger = spdlog::rotating_logger_mt("ttt2", "rotating.log", 1024*1024*300, 100);
    spdlog::set_default_logger(logger);
    logger->set_pattern("%# %@[%Y-%m-%d %H:%M:%S.%e][%^%l%$][%t] %!, %v");
    logger->set_level(spdlog::level::info);
    logger->info("1111111111");
    logger->warn("22222222222");
    logger->debug("333333333333");
    SPDLOG_LOGGER_INFO(logger, "324324djlsl");
    //spdlog::set_default_logger(logger);
    // spdlog::register_logger(logger);
    // spdlog::info("welcome to spdlog");
    // spdlog::set_pattern("[%# %@][%Y-%m-%d %H:%M:%S.%e][%^%l%$][%t] %!, %v");
    // spdlog::info("Positional args are {1} {0}..", "too", "supported");
    // spdlog::set_level(spdlog::level::debug);
    // spdlog::debug("This message should be displayed..");    
    SPDLOG_TRACE("Some trace message with param {}", 42);
    //auto logger = spdlog::basic_logger_mt("basic_logger", "basic_t1.log");

    return 0;
}