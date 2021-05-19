///////////
/////缓存线程安全
//////////
#ifndef _SIMPLE_CACHE_HPP_
#define _SIMPLE_CACHE_HPP_

#include <unordered_map>
#include "../multi_thread/thread_lock.hpp"
#include "time_function.hpp"

namespace su
{
template<typename T>
struct ItemValue
{
    T val;
    unsigned int expire_time;
};

template<typename T_KEY, typename T_VAL, int TTL=300>
class SimpleCacheMap
{
private:
    mutable LockMutex m_mutex;
    std::unordered_map<T_KEY, ItemValue<T_VAL> > m_cache;

public:
    SimpleCacheMap()
    {
        Clear();
    }
    ~SimpleCacheMap()
    {
        Clear();
    }

public:
    int SetCache(const T_KEY& a_key, const T_VAL& a_val)
    {
        MUTEX_GUARD(m_mutex);
        typename std::unordered_map<T_KEY, ItemValue<T_VAL> >::iterator itor = m_cache.find(a_key);
        if (itor != m_cache.end())
        {
            itor->second.val = a_val;
            itor->second.expire_time = (unsigned int)(TTL+second_time());
            return 0;
        }
        std::pair<typename std::unordered_map<T_KEY, ItemValue<T_VAL> >::iterator, bool> pr = 
            m_cache.insert(typename std::unordered_map<T_KEY, ItemValue<T_VAL> >::value_type(a_key, ItemValue<T_VAL>()));
        pr.first->second.val = a_val;
        pr.first->second.expire_time = (unsigned int)(TTL+second_time());
        return 0;
    }

    int GetCache(const T_KEY& a_key, T_VAL& a_val)
    {
        MUTEX_GUARD(m_mutex);
        typename std::unordered_map<T_KEY, ItemValue<T_VAL> >::iterator itor = m_cache.find(a_key);
        if (itor == m_cache.end())
        {
            return 1; //////没缓存
        }
        if (itor->second.expire_time < (unsigned int)second_time())
        {
            m_cache.erase(itor);
            return 2;////////////过期
        }
        a_val = itor->second.val;
        return 0;
    }

    void DelCache(const T_KEY& a_key)
    {
        MUTEX_GUARD(m_mutex);
        typename std::unordered_map<T_KEY, ItemValue<T_VAL> >::iterator itor = m_cache.find(a_key);
        if (itor != m_cache.end())
            m_cache.erase(itor);
    }

    void Clear()
    {
        m_cache.clear();
    }

    int GetTTL()
    {
        return TTL;
    }

    unsigned int GetCacheSize()
    {
        return m_cache.size();
    }
};

}


#endif