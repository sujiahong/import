#ifndef _TIME_FUNCTION_HPP_
#define _TIME_FUNCTION_HPP_

#include <ctime>
#include<sys/time.h>

#define DAY_SECONDS 86400
#define HOUR_SECONDS 3600

namespace su
{
//////返回当前时间戳  秒
static unsigned long long SecondTime()
{
    long int t = time(0);
    if (t != -1)
        return (unsigned long long)t;
    else
        return 0;
}

//////返回当前时间戳  毫秒
static unsigned long long MilliTime()
{
    struct timeval tv;
    if (gettimeofday(&tv, 0) == 0)
        return tv.tv_sec*1000+tv.tv_usec/1000;
    else
        return 0;
}

//////返回当前时间戳  微秒
static unsigned long long MicroTime()
{
    struct timeval tv;
    if (gettimeofday(&tv, 0) == 0)
        return tv.tv_sec*1000000+tv.tv_usec;
    else
        return 0;
}

//////返回当前时间戳  纳秒
static unsigned long long NanoTime()
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
static int GetTimeZone(unsigned long long a_cur_time=0)
{
    if (a_cur_time == 0)
        a_cur_time = SecondTime();
    struct tm tm_local;
    localtime_r((long*)&a_cur_time, &tm_local);
    struct tm tm_gm;
    gmtime_r((long*)&a_cur_time, &tm_gm);
    //std::cout << "hour="<<tm_gm.tm_hour <<" sec="<<tm_gm.tm_sec<<std::endl;
    return tm_local.tm_hour - tm_gm.tm_hour;
}

////返回每天零点时间戳 秒 
/////参数a_cur_time==0时，自动获取当前时间戳零点 东八区
static unsigned long long ZeroSecTime(unsigned long long a_cur_time=0)
{
    if (a_cur_time == 0)
        a_cur_time = SecondTime();
    int time_zone = GetTimeZone(a_cur_time);
    return a_cur_time - (a_cur_time + time_zone*HOUR_SECONDS)%DAY_SECONDS;
}

/// 返回当前时间是周几
static unsigned int WeekDay(unsigned long long a_cur_time=0)
{
    if (a_cur_time == 0)
        a_cur_time = SecondTime();
    struct tm tm_local;
    localtime_r((long*)&a_cur_time, &tm_local);
    if (tm_local.tm_wday == 0) return 7;
    return (unsigned int)(tm_local.tm_wday);
}

/// @brief 返回日期对应的时间戳
/// a_date  格式 20230506  
/// @return  当前时间戳
static unsigned long long DateToTimeStamp(unsigned int a_date)
{
	unsigned int year = a_date/10000;
	unsigned int month = (a_date-year*10000)/100;
	unsigned int day = (a_date-year*10000-month*100);
    
}

}

#endif