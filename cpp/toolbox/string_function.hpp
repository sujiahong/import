
#ifndef _STRING_FUNCTION_HPP_
#define _STRING_FUNCTION_HPP_

#include <string>
#include <vector>
#include<sstream>
#include <list>
#include <map>
#include <set>

namespace su
{
//////number为整数，浮点数
template<typename T>
T String2Number(const std::string& a_str)
{
    std::stringstream ss;
    ss<<a_str;
    T tmp;
    ss>>tmp;
    return tmp;
}

template<typename T>
std::string Number2String(T a_n)
{
    std::stringstream ss;
    ss<<a_n;
    return ss.str();
}

////字符串切割
template<typename T>
void Split(const std::string& a_str, const std::string& a_delim, std::vector<T>& a_vec)
{
  std::string::size_type pos1, pos2;
  pos2 = a_str.find(a_delim);
  pos1 = 0;
  while(std::string::npos != pos2)
  {
    a_vec.push_back(String2Number<T>(a_str.substr(pos1, pos2-pos1)));
 
    pos1 = pos2 + a_delim.size();
    pos2 = a_str.find(a_delim, pos1);
  }
  if(pos1 != a_str.length())
    a_vec.push_back(String2Number<T>(a_str.substr(pos1)));
}
template<typename KT, typename VT>
void SplitToMap(const std::string& a_str, const std::string& a_d1, const std::string& a_d2, std::map<KT, VT>& a_map)
{
    std::vector<std::string> vec;
    Split(a_str, a_d1, vec);
    std::string::size_type pos;
    std::string key, val;
    for (unsigned int i = 0; i < vec.size(); ++i)
    {
        pos = vec[i].find(a_d2);
        key = vec[i].substr(0, pos);
        val = vec[i].substr(pos+a_d2.size());
        a_map.insert(std::make_pair(String2Number<KT>(key), String2Number<VT>(val)));
    }
}

template<class K, class V>
std::string Map2String(const std::map<K, V>& obj_msp, int max_disp_count = 5) {
    std::ostringstream oss;
    oss << "{";
    int i = 0;
    typename std::map<K, V>::const_iterator it = obj_msp.begin();
    while (it != obj_msp.end()) {
        if (i >= max_disp_count) {
            break;
        }
        if (i > 0) {
            oss << ", ";
        }
        oss << it->first << ":" << it->second;
        i++;
        it++;
    }
    if (obj_msp.size() > max_disp_count) {
        oss << ", (" << obj_msp.size() - max_disp_count << " more)...";
    }
    oss << "}";
    return oss.str();
}
template<typename KeyT, typename ValueT>
std::string Map2String(const std::map<KeyT,ValueT>& a_map, const std::string a_map_split= ",", const std::string a_pair_split = ":")
{
    std::stringstream ss;
    typename std::map<KeyT, ValueT>::const_iterator iter = a_map.begin();
    for(; iter != a_map.end(); ++iter)
    {
        if(iter != a_map.begin())
        {
            ss << a_map_split;
        }
        ss << iter->first;
        ss << a_pair_split;
        ss << iter->second;
    }
    return ss.str();
}

template<class T>
std::string Vec2String(const std::vector<T>& obj_vec, int max_disp_count = 5) {
    std::ostringstream oss;
    oss << "[";
    for (int i = 0; i < obj_vec.size(); i++) {
        if (i >= max_disp_count) {
            break;
        }
        if (i > 0) {
            oss << ", ";
        }
        oss << obj_vec[i];
    }
    if (obj_vec.size() > max_disp_count) {
        oss << ", (" << obj_vec.size() - max_disp_count << " more)...";
    }
    oss << "]";
    return oss.str();
}

template<class T>
std::string List2String(const std::list<T>& obj_list, int max_disp_count = 5) {
    std::ostringstream oss;
    oss << "[";
    int i = 0;
    typename std::list<T>::const_iterator it = obj_list.begin();
    while (it != obj_list.end()) {
        if (i >= max_disp_count) {
            break;
        }
        if (i > 0) {
            oss << ", ";
        }
        oss << *it;
        i++;
        it++;
    }
    if (obj_list.size() > max_disp_count) {
        oss << ", (" << obj_list.size() - max_disp_count << " more)...";
    }
    oss << "]";
    return oss.str();
}

template<class T>
std::string Set2String(const std::set<T>& obj_set, int max_disp_count = 5) {
    std::ostringstream oss;
    oss << "{";
    int i = 0;
    typename std::set<T>::const_iterator it = obj_set.begin();
    while (it != obj_set.end()) {
        if (max_disp_count > 0 && i >= max_disp_count) {
            break;
        }
        if (i > 0) {
            oss << ", ";
        }
        oss << *it;
        i++;
        it++;
    }
    if (max_disp_count > 0 && obj_set.size() > max_disp_count) {
        oss << ", (" << obj_set.size() - max_disp_count << " more)...";
    }
    oss << "}";
    return oss.str();
}
template<class T>
std::string Set2String(const std::set<T>& a_set) {
    std::ostringstream oss;
    oss << "{";
    typename std::set<T>::const_iterator it = a_set.begin();
    while (it != a_set.end()) {
        oss << *it;
        it++;
        if(it != a_set.end())
            oss << ",";
    }
    oss << "}";
    return oss.str();
}

template<class T>
std::string Object2String(const T& obj)/////这个怎么实现
{
    return "";
}

}


#endif