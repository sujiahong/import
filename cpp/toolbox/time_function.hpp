#ifndef _TIME_FUNCTION_HPP_
#define _TIME_FUNCTION_HPP_

#include <ctime>
#include<sys/time.h>
#include <string>
#include <cstring>
#include <chrono>

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

//////返回当前时间戳  毫秒
static unsigned long long MilliTimeCR()
{
    using namespace std::chrono;
    return duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
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
static unsigned long long MicroTimeCR()
{
    using namespace std::chrono;
#if __cplusplus >= 202002L
    return duration_cast<microseconds>(utc_clock::now().time_since_epoch()).count();
#else
    return duration_cast<microseconds>(system_clock::now().time_since_epoch()).count();
#endif
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
    time_t raw_time = static_cast<time_t>(a_cur_time);
    struct tm tm_local;
    localtime_r(&raw_time, &tm_local);
    struct tm tm_gm;
    gmtime_r(&raw_time, &tm_gm);
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
	unsigned int day =  a_date % 100;

    struct tm tm_local = {0};
    tm_local.tm_year = year-1900;
    tm_local.tm_mon = month-1;
    tm_local.tm_mday = day;

    time_t t = mktime(&tm_local);
    if (t == -1) 
        throw std::runtime_error("Invalid date");
    return static_cast<unsigned long long>(t);
}
// 输入 a_cur_time 当前时间戳 秒
//返回 年月 例如：202305 整数
static unsigned int DateYearMonth(unsigned long long a_cur_time=0)
{
    if (a_cur_time == 0)
        a_cur_time = SecondTime();
    struct tm tm_local;
    localtime_r((long*)&a_cur_time, &tm_local);
    return (tm_local.tm_year+1900)*100+tm_local.tm_mon+1;
}
// 输入 a_cur_time 当前时间戳 秒
//返回 年月 例如：202305 字符串
static std::string DateYearMonthString(unsigned long long a_cur_time=0)
{
    if (a_cur_time == 0)
        a_cur_time = SecondTime();
    thread_local struct tm tm_local;
    memset(&tm_local, 0, sizeof(tm_local));
    localtime_r((long*)&a_cur_time, &tm_local);
    thread_local char dm[16] = {0};
    memset(dm, 0, sizeof(dm));
    strftime(dm, sizeof(dm), "%Y%m", &tm_local);
    std::string dm_str(dm);
    return dm_str;
}

// 输入 a_cur_time 当前时间戳 秒
/// @return 年月日 例：20230506 整数
static unsigned int DateYearMonthDay(unsigned long long a_cur_time=0)
{
    if (a_cur_time == 0)
        a_cur_time = SecondTime();
    struct tm tm_local;
    localtime_r((long*)&a_cur_time, &tm_local);
    return (tm_local.tm_year+1900)*10000+(tm_local.tm_mon+1)*100+tm_local.tm_mday;
}
// 输入 a_cur_time 当前时间戳 秒
/*

操作	时钟周期数 (x86)
普通栈变量访问	1
TLS 变量访问	3-5
全局变量+锁访问	20-100
*/
/// @return 年月日 例：20230506 字符串
static std::string DateYearMonthDayString(unsigned long long a_cur_time=0)
{
    if (a_cur_time == 0)
        a_cur_time = SecondTime();
    thread_local struct tm tm_local;
    memset(&tm_local, 0, sizeof(tm_local));
    localtime_r((long*)&a_cur_time, &tm_local);
    thread_local char dm[10];
    memset(dm, 0, sizeof(dm));
    strftime(dm, sizeof(dm), "%Y%m%d", &tm_local);
    std::string dm_str(dm);
    return dm_str;
}


}

#endif