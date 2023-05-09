#ifndef _RANDOM_DISORDER_HPP_
#define _RANDOM_DISORDER_HPP_

#include <vector>
#include <unordered_set>
#include <iostream>
#include <cstdlib>
#include "../toolbox/time_function.hpp"


namespace su
{

/////产生[min--max]范围随机数
int RangeRandom(int a_min, int a_max)
{
    if (a_max < a_min)
        return 0;
    return rand() % (a_max-a_min+1) + a_min;
}

/////1-n数，m元排列,,, floyd 随机取样
void FloydSample(int n, int m, std::vector<int>& a_vec)
{
    ////std::vector<unsigned int>().swap(a_vec);/////释放内存
    a_vec.resize(m);///////设置数组大小
    a_vec.clear();////////清除数据,并不清除内存，只是改变大小
    std::unordered_set<int> unod_set(n+m);
    int r = 0;
    for (int j = n - m + 1; j <= n; ++j)
    {
        r = RangeRandom(1, j);
        //std::cout << "r="<<r <<" j="<<j<<" capacity="<<a_vec.capacity()<<std::endl;
        if (unod_set.find(r) == unod_set.end())
        {
            unod_set.insert(r);
            a_vec.push_back(r);
        }
        else
        {
            unod_set.insert(j);
            a_vec.push_back(j);
        }
    }
}

template<typename T>
void DisorderArray(std::vector<T>& a_vec)
{
    T tmp;
    unsigned int rand_num = 0;
    srand(SecondTime());
    for(unsigned int i = 0; i < a_vec.size(); ++i)
    {
        rand_num = RangeRandom(i, a_vec.size()-1);
        //std::cout << "rand_num="<<rand_num <<" i="<<i<<std::endl;
        tmp = a_vec[i];
        a_vec[i] = a_vec[rand_num];
        a_vec[rand_num] = tmp;
    }
}

//////数组乱序////有问题,会出现有的排列出现的概率高一点
template<typename T>
void DisorderArrayV2(std::vector<T>& a_vec)
{
    T tmp;
    unsigned int rand_num = 0;
    srand(SecondTime());
    for(unsigned int i = 0; i < a_vec.size(); ++i)
    {
        rand_num = RangeRandom(0, a_vec.size()-1);
        tmp = a_vec[i];
        a_vec[i] = a_vec[rand_num];
        a_vec[rand_num] = tmp;
    }
}

}


#endif