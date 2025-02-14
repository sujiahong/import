/*字符串，字符指针计算hash值*/
#include <cstdint>
#include <string>
#include <chrono>

namespace su
{
// DJB2哈希算法（经典字符串哈希算法）
uint32_t djb2_hash(const std::string& str, uint32_t seed = 5381) 
{
    uint32_t hash = seed;
    for (char c : str) {
        hash = ((hash << 5) + hash) + c; // hash * 33 + c
    }
    return hash;
}

// FNV-1a哈希算法（适用于小数据）
uint32_t fnv1a_hash(const std::string& str, uint32_t seed = 0x811C9DC5) 
{
    const uint32_t prime = 0x01000193; // 16777619
    uint32_t hash = seed;
    for (char c : str) {
        hash ^= static_cast<uint32_t>(c);
        hash *= prime;
    }
    return hash;
}

// 组合哈希（结合时间和上述算法）
uint64_t combined_hash(const std::string& str) 
{
    auto time_hash = static_cast<uint64_t>(std::chrono::system_clock::now().time_since_epoch().count());
    return (time_hash << 32) | djb2_hash(str);
}


inline static uint64_t load_bytes(const char* p, int n)
{
    uint64_t result = 0;
    --n;
    do
        result = (result << 8) + static_cast<unsigned char>(p[n]);
    while (--n >= 0);
    return result;
}
inline static uint64_t shift_mix(uint64_t v) 
{
    return v ^ (v >> 47);
}
static const uint64_t f_m = (((uint64_t) 0xc6a4a793UL) << 32UL) + (uint64_t) 0x5bd1e995UL;
/// @brief 通过指针计算hash值 
uint64_t supcHash(const void* ptr, uint32_t len, uint64_t seed = 0xc70f6907UL)
{
    const char* buf = static_cast<const char*>(ptr);
    const uint32_t len_align = len & ~0x7;
    const char* buf_end = buf + len_align;
    uint64_t hash = seed ^ (len * f_m);
    for (const char* p = buf; p < buf_end; p += 8)
    {
        const uint64_t td = shift_mix(*(const uint64_t*)p) * f_m;
        hash ^= td;
        hash *= f_m;
    }
    if ((len & 0x7) != 0)
    {
        const uint64_t td = load_bytes(buf_end, len & 0x7);
        hash ^= td;
        hash *= f_m;
    }
    hash = shift_mix(hash) * f_m;
    hash = shift_mix(hash);
    return hash;
}

}