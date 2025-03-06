
#ifndef _STRING_FUNCTION_HPP_
#define _STRING_FUNCTION_HPP_

#include <string>
#include <vector>
#include <sstream>
#include <list>
#include <map>
#include <set>
#include <unordered_map>
#include <type_traits>

namespace su
{
//////number为整数，浮点数
template<typename T>
T String2Number(const std::string& a_str)
{
    std::stringstream ss;
    ss<<a_str;
    T tmp{};
    ss >> tmp;
    if (ss.fail())
    {
        return 0;
    }
    // char remain;
    // if (ss >> remain)
    // {
    //     throw std::invalid_argument("Trailing characters in: " + a_str);
    // }
    return tmp;
}
// 特化字符串类型
template<>
std::string String2Number<std::string>(const std::string& a_str)
{
    return a_str;
}

template<typename T>
std::string Number2String(T a_n)
{
    std::stringstream ss;
    ss<<a_n;
    return ss.str();
}

template<typename T>
std::string Number2StringSTL(T a_n)
{
    return std::to_string(a_n);
}

////字符串切割
template<typename T>
void Split(const std::string& a_str, const std::string& a_delim, std::vector<T>& a_vec)
{
    a_vec.clear();
    if (a_str.empty() || a_delim.empty())
        return;
    std::string::size_type pos1, pos2;
    pos1 = 0;
    while(std::string::npos != (pos2 = a_str.find(a_delim, pos1)))
    {
        a_vec.push_back(String2Number<T>(a_str.substr(pos1, pos2-pos1)));
        pos1 = pos2 + a_delim.size();
    }
    if(pos1 != a_str.length())
        a_vec.push_back(String2Number<T>(a_str.substr(pos1)));
}
template<typename KT, typename VT>
void SplitToMap(const std::string& a_str, const std::string& a_d1, const std::string& a_d2, std::map<KT, VT>& a_map)
{
    a_map.clear();
    if (a_str.empty() || a_d1.empty() || a_d2.empty())
        return;
    std::vector<std::string> vec;
    Split(a_str, a_d1, vec);
    std::string::size_type pos;
    std::string key, val;
    for (const auto& tmp_str : vec)////范围for语句
    {
        pos = tmp_str.find(a_d2);
        if (pos != std::string::npos)
        {
            key = tmp_str.substr(0, pos);
            val = tmp_str.substr(pos+a_d2.size());
            a_map.insert(std::make_pair(String2Number<KT>(key), String2Number<VT>(val)));
        }
    }
}

// template<class K, class V>
// std::string Map2String(const std::map<K, V>& obj_msp, int max_disp_count = 5) {
//     std::ostringstream oss;
//     oss << "{";
//     int i = 0;
//     typename std::map<K, V>::const_iterator it = obj_msp.begin();
//     while (it != obj_msp.end()) {
//         if (i >= max_disp_count) {
//             break;
//         }
//         if (i > 0) {
//             oss << ", ";
//         }
//         oss << it->first << ":" << it->second;
//         i++;
//         it++;
//     }
//     if (obj_msp.size() > max_disp_count) {
//         oss << ", (" << obj_msp.size() - max_disp_count << " more)...";
//     }
//     oss << "}";
//     return oss.str();
// }
// template<typename KeyT, typename ValueT>
// std::string Map2String(const std::map<KeyT,ValueT>& a_map, const std::string a_map_split= ",", const std::string a_pair_split = ":")
// {
//     std::stringstream ss;
//     typename std::map<KeyT, ValueT>::const_iterator iter = a_map.begin();
//     for(; iter != a_map.end(); ++iter)
//     {
//         if(iter != a_map.begin())
//         {
//             ss << a_map_split;
//         }
//         ss << iter->first;
//         ss << a_pair_split;
//         ss << iter->second;
//     }
//     return ss.str();
// }

// template<class T>
// std::string Vec2String(const std::vector<T>& obj_vec, int max_disp_count = 5) {
//     std::ostringstream oss;
//     oss << "[";
//     for (int i = 0; i < obj_vec.size(); i++) {
//         if (i >= max_disp_count) {
//             break;
//         }
//         if (i > 0) {
//             oss << ", ";
//         }
//         oss << obj_vec[i];
//     }
//     if (obj_vec.size() > max_disp_count) {
//         oss << ", (" << obj_vec.size() - max_disp_count << " more)...";
//     }
//     oss << "]";
//     return oss.str();
// }

// template<class T>
// std::string List2String(const std::list<T>& obj_list, int max_disp_count = 5) {
//     std::ostringstream oss;
//     oss << "[";
//     int i = 0;
//     typename std::list<T>::const_iterator it = obj_list.begin();
//     while (it != obj_list.end()) {
//         if (i >= max_disp_count) {
//             break;
//         }
//         if (i > 0) {
//             oss << ", ";
//         }
//         oss << *it;
//         i++;
//         it++;
//     }
//     if (obj_list.size() > max_disp_count) {
//         oss << ", (" << obj_list.size() - max_disp_count << " more)...";
//     }
//     oss << "]";
//     return oss.str();
// }

// template<class T>
// std::string Set2String(const std::set<T>& obj_set, int max_disp_count = 5) {
//     std::ostringstream oss;
//     oss << "{";
//     int i = 0;
//     typename std::set<T>::const_iterator it = obj_set.begin();
//     while (it != obj_set.end()) {
//         if (max_disp_count > 0 && i >= max_disp_count) {
//             break;
//         }
//         if (i > 0) {
//             oss << ", ";
//         }
//         oss << *it;
//         i++;
//         it++;
//     }
//     if (max_disp_count > 0 && obj_set.size() > max_disp_count) {
//         oss << ", (" << obj_set.size() - max_disp_count << " more)...";
//     }
//     oss << "}";
//     return oss.str();
// }
// template<class T>
// std::string Set2String(const std::set<T>& a_set) {
//     std::ostringstream oss;
//     oss << "{";
//     typename std::set<T>::const_iterator it = a_set.begin();
//     while (it != a_set.end()) {
//         oss << *it;
//         it++;
//         if(it != a_set.end())
//             oss << ",";
//     }
//     oss << "}";
//     return oss.str();
// }

template<typename T>
struct is_map : std::false_type {};

template<typename K, typename V, typename Compare, typename Alloc>
struct is_map<std::map<K,V, Compare, Alloc> > : std::true_type {};

// 特化版本识别 std::unordered_map
template <typename Key, typename T, typename Hash, typename KeyEqual, typename Alloc>
struct is_map<std::unordered_map<Key, T, Hash, KeyEqual, Alloc>> : std::true_type {};

template<typename T>
inline constexpr bool is_map_v = is_map<T>::value;

///自定义容器类型不支持
template<typename Container>
std::string Container2String(const Container& c, const std::string& delim=", ", 
    const std::string& pair_delim = "", int max_disp = 100) -> std::enable_if_t<!is_map_v<Container>, std::string>
{
    std::ostringstream oss;
    oss << "[";
    int count = 0;
    for (auto it = c.begin(); it != c.end() && count < max_disp; ++it, ++count)
    {
        if (count > 0) oss << delim;
        oss << *it;
    }
    if (count < c.size())
        oss << " ... (" << c.size() - count << " more)";
    oss << "]";
    return oss.str();
}

template<typename K, typename V>
std::string Container2String(const std::map<K,V>& m, const std::string& pair_delim=":", 
    const std::string& item_delim=", ", int max_disp = 100)
{
    std::ostringstream oss;
    oss << "{";
    int count = 0;
    for (auto it = m.begin(); it != m.end() && count < max_disp; ++it, ++count) {
        if (count > 0) oss << item_delim;
        oss << it->first << pair_delim << it->second;
    }
    if (count < m.size()) {
        oss << " ... (" << m.size() - count << " more)";
    }
    oss << "}";
    return oss.str();
}
template<typename K, typename V>
std::string Container2String(const std::unordered_map<K,V>& m, const std::string& pair_delim=":", 
    const std::string& item_delim=", ", int max_disp = 100)
{
    std::ostringstream oss;
    oss << "{";
    int count = 0;
    for (auto it = m.begin(); it != m.end() && count < max_disp; ++it, ++count) {
        if (count > 0) oss << item_delim;
        oss << it->first << pair_delim << it->second;
    }
    if (count < m.size()) {
        oss << " ... (" << m.size() - count << " more)";
    }
    oss << "}";
    return oss.str();
}

/////替换字符串A中所有字串B为子串C
std::string replace_all(std::string origin_str, const std::string& old_value, const std::string& new_value)
{
    if (origin_str.empty())
        return "";
    if (new_value.empty() || old_value.empty())
        return origin_str;
    size_t pos = 0;
    while((pos = origin_str.find(old_value, pos)) != std::string::npos)
    {
        origin_str.replace(pos, old_value.length(), new_value);
        pos += new_value.length();
    }
    return origin_str;
}
std::string replace_all_fast(const std::string& origin_str, const std::string& old_value, const std::string& new_value)
{
    if (origin_str.empty())
        return "";
    if (new_value.empty() || old_value.empty())
        return origin_str;
    const size_t str_len = origin_str.length();
    const size_t old_len = old_value.length();
    std::vector<size_t> positions;
    positions.reserve(std::min(str_len/old_len, size_t(128)));
    size_t pos = 0;
    
    while ((pos = origin_str.find(old_value, pos)) != std::string::npos)
    {
        positions.push_back(pos);
        pos += old_len;
    }
    if (positions.empty()) return origin_str;
    const size_t new_len = new_value.length();
    const size_t total_len = str_len + (new_len - old_len) * positions.size();
    std::string result;
    result.reserve(total_len);
    size_t last = 0;
    for (size_t ps : positions)
    {
        result.append(origin_str, last, ps - last);
        result.append(new_value);
        last = ps + old_len;
    }
    result.append(origin_str.substr(last));
    return result;
}

}


#endif