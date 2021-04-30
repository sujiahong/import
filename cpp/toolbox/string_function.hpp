
#ifndef _STRING_FUNCTION_HPP_
#define _STRING_FUNCTION_HPP_

#include <string>
#include <vector>
#include<sstream>

namespace su
{
//////number为整数，浮点数
template<typename T>
T string_to_number(const std::string& a_str)
{
    std::stringstream ss;
    ss<<a_str;
    T tmp;
    ss>>tmp;
    return tmp;
}

template<typename T>
std::string number_to_string(T a_n)
{
    std::stringstream ss;
    ss<<a_n;
    return ss.str();
}

////字符串切割
template<typename T>
void split(const std::string& a_str, const std::string& a_delim, std::vector<T>& a_vec)
{
  std::string::size_type pos1, pos2;
  pos2 = a_str.find(a_delim);
  pos1 = 0;
  while(std::string::npos != pos2)
  {
    a_vec.push_back(string_to_number<T>(a_str.substr(pos1, pos2-pos1)));
 
    pos1 = pos2 + a_delim.size();
    pos2 = a_str.find(a_delim, pos1);
  }
  if(pos1 != a_str.length())
    a_vec.push_back(string_to_number<T>(a_str.substr(pos1)));
}

}


#endif