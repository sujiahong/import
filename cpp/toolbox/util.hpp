#ifndef _UTIL_HPP_
#define _UTIL_HPP_

#include <cmath>

namespace su{
    ////////平方根，，只能float  double有问题
    static float sqrt_v1(float a_n)////使用时需要求倒数
    {
        float half = 0.5f * a_n;
        int i = *(int*)&a_n;
        i = 0x5f3759df-(i>>1);
        a_n = *(float*)&i;
        a_n = a_n*(1.5f - half*a_n*a_n);
        return a_n;
    }
    static float sqrt_v2(float a_n)
    {
        float tmp;
        float half = 0.5f * a_n;
        tmp = a_n;
        long i = *(long*)&tmp;
        i = 0x5f3759df-(i>>1);
        tmp = *(float*)&i;
        tmp = tmp*(1.5f - half*tmp*tmp);
        tmp = tmp*(1.5f - half*tmp*tmp);
        return a_n*tmp;
    }
    /// @brief 小数的二进制有时不能精确，可能造成精度丢失，如果需要进行浮点运算，不要直接用float或者double进行加减乘除
    /// @param original 
    /// @return 
    long Double2Int64(double original)////保证进度不会丢失
    {
	    long temp = static_cast<long>(original);
	    double delta = original - static_cast<double>(temp);
        std::cout<<" delta="<<delta<<" "<<static_cast<double>(temp)<<" "<<temp<<std::endl;
	    if (std::abs(delta) > 0.999999) // 改进精度判断阈值
	    {
	    	if (0 < delta)
	    	{
	    		temp++;
	    	}
	    	else
	    	{
	    		temp--;
	    	}
	    }
	    return temp;
    }
}

#endif