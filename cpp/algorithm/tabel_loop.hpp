
#ifndef _TABEL_LOOP_HPP_
#define _TABEL_LOOP_HPP_

#include <set>
#include <vector>
#include <functional>

namespace su
{

/*
a_container   [in] 数据容器
a_start       [in out] 传入开始位置，传出下一次开始位置
a_out_vec     [out]  输出列表
a_limit_num   [in]   列表数量限制
a_condition_func [in] 函数指针，限制条件
*/
template <typename T> //////////////T 只支持简单类型，不支持组合类型
void GetLoopSetData(std::set<T> &a_containor, T& a_start, std::vector<T> &a_out_vec, const unsigned int a_limit_num, bool (*a_condition_func)(const T&))
{
    if (a_containor.size() <= 0)
    {
        return;
    }
    typename std::set<T>::iterator itor = a_containor.begin();
    itor = a_containor.find(a_start);
    if (itor == a_containor.end())
    {
        itor = a_containor.begin();
    }
    while (1)
    {
        if (a_condition_func && a_condition_func(*itor))
        {
            ++itor;
            if (itor == a_containor.end())
            {
                itor = a_containor.begin();
                if (a_out_vec.size() > 0)
                {
                    if (a_out_vec[0] == *itor)
                    {
                        a_start = *itor;
                        break;
                    }
                    else
                        continue;
                }
                else
                {
                    break; //////一个都没有
                }
            }
            continue;
        }
        if (a_out_vec.size() > 0 && a_out_vec[0] == *itor)
        {
            a_start = *itor;
            break;
        }
        a_out_vec.push_back(*itor);
        if (a_out_vec.size() >= a_limit_num)
        {
            ++itor;
            if (itor == a_containor.end())
                itor = a_containor.begin();
            a_start = *itor;
            break;
        }
        ++itor;
        if (itor == a_containor.end())
            itor = a_containor.begin();
        if (a_out_vec.size() > 0 && a_out_vec[0] == *itor)
        {
            a_start = *itor;
            break;
        }
    }
}

template <typename T> //////////////T 只支持简单类型，不支持组合类型
void GetLoopSetData(std::set<T> &a_containor, T& a_start, std::vector<T> &a_out_vec, const unsigned int a_limit_num, std::function<bool(const T&)> a_condition_func)
{
    if (a_containor.size() <= 0)
    {
        return;
    }
    typename std::set<T>::iterator itor = a_containor.begin();
    itor = a_containor.find(a_start);
    if (itor == a_containor.end())
    {
        itor = a_containor.begin();
    }
    while (1)
    {
        if (a_condition_func && a_condition_func(*itor))
        {
            ++itor;
            if (itor == a_containor.end())
            {
                itor = a_containor.begin();
                if (a_out_vec.size() > 0)
                {
                    if (a_out_vec[0] == *itor)
                    {
                        a_start = *itor;
                        break;
                    }
                    else
                        continue;
                }
                else
                {
                    break; //////一个都没有
                }
            }
            continue;
        }
        if (a_out_vec.size() > 0 && a_out_vec[0] == *itor)
        {
            a_start = *itor;
            break;
        }
        a_out_vec.push_back(*itor);
        if (a_out_vec.size() >= a_limit_num)
        {
            ++itor;
            if (itor == a_containor.end())
                itor = a_containor.begin();
            a_start = *itor;
            break;
        }
        ++itor;
        if (itor == a_containor.end())
            itor = a_containor.begin();
        if (a_out_vec.size() > 0 && a_out_vec[0] == *itor)
        {
            a_start = *itor;
            break;
        }
    }
}

/*
a_container   [in] 数据容器
a_start       [in out] 传入开始位置，传出下一次开始位置
a_out_vec     [out]  输出列表
a_limit_num   [in]   列表数量限制
*/
template <typename T> //////////////T 只支持简单类型，不支持组合类型
void GetLoopVectorData(std::vector<T> &a_containor, unsigned int &a_start_subscript, std::vector<T> &a_out_vec, const unsigned int a_limit_num, bool (*a_condition_func)(const T &))
{
    if (a_containor.size() <= 0)
    {
        return;
    }
    unsigned int cur_subpt = a_start_subscript;
    if (cur_subpt >= a_containor.size())
        cur_subpt = 0;
    while (1)
    {
        if (a_condition_func && a_condition_func(a_containor[cur_subpt]))
        {
            ++cur_subpt;
            if (cur_subpt >= a_containor.size())
            {
                cur_subpt = 0;
                if (a_out_vec.size() > 0)
                {
                    if (a_out_vec[0] == a_containor[cur_subpt])
                    {
                        a_start_subscript = cur_subpt;
                        break;
                    }
                    else
                        continue;
                }
                else
                {
                    break; //////一个都没有
                }
            }
            continue;
        }
        if (a_out_vec.size() > 0 && a_out_vec[0] == a_containor[cur_subpt])
        {
            a_start_subscript = cur_subpt;
            break;
        }
        a_out_vec.push_back(a_containor[cur_subpt]);
        if (a_out_vec.size() >= a_limit_num)
        {
            ++cur_subpt;
            if (cur_subpt >= a_containor.size())
                cur_subpt = 0;
            a_start_subscript = cur_subpt;
            break;
        }
        ++cur_subpt;
        if (cur_subpt >= a_containor.size())
            cur_subpt = 0;
        if (a_out_vec.size() > 0 && a_out_vec[0] == a_containor[cur_subpt])
        {
            a_start_subscript = cur_subpt;
            break;
        }
    }
}

}

#endif /////循环从列表读数据