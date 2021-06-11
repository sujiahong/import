////////
////uuid生成器
////////

#ifndef _UUID_HPP_
#define _UUID_HPP_

#include "time_function.hpp"
#include "../thread_lock.hpp"

namespace su
{

static unsigned long long g_last_time = 0;
su::LockMutex last_time_mutex;


//////单进程下生成uuid，保证id不重复
static unsigned long long sp_uuid()
{
    unsigned long long time = nano_time();
    MUTEX_GUARD(last_time_mutex);
    if (time > g_last_time)
    {
        g_last_time = time;
    }
    else
    {
        ++g_last_time;
    }
    return g_last_time;
}


}


#endif