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
    SimpleCacheMap();
    ~SimpleCacheMap();

public:
    int SetCache(const T_KEY& a_key, const T_VAL& a_val);

    int GetCache(const T_KEY& a_key, T_VAL& a_val);

    void DelCache(const T_KEY& a_key);

    void Clear();

    int GetTTL();

    unsigned int GetCacheSize();
};

template<typename T_KEY, typename T_VAL, int TTL>
SimpleCacheMap<T_KEY, T_VAL, TTL>::SimpleCacheMap()
{
    Clear();
}

template<typename T_KEY, typename T_VAL, int TTL>
SimpleCacheMap<T_KEY, T_VAL, TTL>::~SimpleCacheMap()
{
    Clear();
}

template<typename T_KEY, typename T_VAL, int TTL>
int SimpleCacheMap<T_KEY, T_VAL, TTL>::SetCache(const T_KEY& a_key, const T_VAL& a_val)
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

template<typename T_KEY, typename T_VAL, int TTL>
int SimpleCacheMap<T_KEY, T_VAL, TTL>::GetCache(const T_KEY& a_key, T_VAL& a_val)
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

template<typename T_KEY, typename T_VAL, int TTL>
void SimpleCacheMap<T_KEY, T_VAL, TTL>::DelCache(const T_KEY& a_key)
{
    MUTEX_GUARD(m_mutex);
    typename std::unordered_map<T_KEY, ItemValue<T_VAL> >::iterator itor = m_cache.find(a_key);
    if (itor != m_cache.end())
        m_cache.erase(itor);
}

template<typename T_KEY, typename T_VAL, int TTL>
void SimpleCacheMap<T_KEY, T_VAL, TTL>::Clear()
{
    m_cache.clear();
}

template<typename T_KEY, typename T_VAL, int TTL>
int SimpleCacheMap<T_KEY, T_VAL, TTL>::GetTTL()
{
    return TTL;
}

template<typename T_KEY, typename T_VAL, int TTL>
unsigned int SimpleCacheMap<T_KEY, T_VAL, TTL>::GetCacheSize()
{
    return m_cache.size();
}

}


#endif