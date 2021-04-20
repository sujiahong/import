#ifndef _TIME_FUNCTION_HPP_
#define _TIME_FUNCTION_HPP_

#include <ctime>
#include<sys/time.h>

#define DAY_SECONDS 86400
#define HOUR_SECONDS 3600

namespace su
{
//////返回当前时间戳  秒
unsigned long second_time()
{
    long int t = time(0);
    if (t != -1)
        return (unsigned long int)t;
    else
        return 0;
}

//////返回当前时间戳  毫秒
unsigned long int milli_time()
{
    struct timeval tv;
    if (gettimeofday(&tv, 0) == 0)
        return tv.tv_sec*1000+tv.tv_usec/1000;
    else
        return 0;
}

//////返回当前时间戳  微秒
unsigned long int micro_time()
{
    struct timeval tv;
    if (gettimeofday(&tv, 0) == 0)
        return tv.tv_sec*1000000+tv.tv_usec;
    else
        return 0;
}

//////返回当前时间戳  纳秒
unsigned long int nano_time()
{
    struct timespec ts;
    if (clock_gettime(CLOCK_REALTIME, &ts) == 0)
        return ts.tv_sec*1000000000+ts.tv_nsec;
    else
        return 0;
}

/*
struct tm {  
    int tm_sec;         // seconds
    int tm_min;         // minutes
    int tm_hour;        // hours
    int tm_mday;        // day of the month 
    int tm_mon;         // month
    int tm_year;        // year
    int tm_wday;        // day of the week
    int tm_yday;        // day in the year
    int tm_isdst;       // daylight saving time  
};
*/
///获取当前时区  小时
int get_time_zone(unsigned long a_cur_time=0)
{
    if (a_cur_time == 0)
        a_cur_time = second_time();
    struct tm tm_local;
    localtime_r((long*)&a_cur_time, &tm_local);
    struct tm tm_gm;
    gmtime_r((long*)&a_cur_time, &tm_gm);
    //std::cout << "hour="<<tm_gm.tm_hour <<" sec="<<tm_gm.tm_sec<<std::endl;
    return tm_local.tm_hour - tm_gm.tm_hour;
}

////返回每天零点时间戳 秒 
/////参数a_cur_time==0时，自动获取当前时间戳零点 东八区
unsigned long zero_sec_time(unsigned long a_cur_time=0)
{
    if (a_cur_time == 0)
        a_cur_time = second_time();
    int time_zone = get_time_zone(a_cur_time);
    return a_cur_time - (a_cur_time + time_zone*HOUR_SECONDS)%DAY_SECONDS;
}



}

#endif