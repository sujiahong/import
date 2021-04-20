

#include "./algorithm/tabel_loop.hpp"
#include "./toolbox/time_function.hpp"
#include "./algorithm/random_disorder.hpp"
#include "./toolbox/string_function.hpp"
#include <set>
#include <vector>
#include <iostream>

bool condition_func(const int& val)
{
    return (val == 3);
}

int main1(int argc, char** argv)
{
    std::set<int> set;
    set.insert(1);
    set.insert(2);
    set.insert(3);
    set.insert(4);
    set.insert(5);
    
    int start = 1;
    std::vector<int> vec;

    for (int s = 1; s < 6; ++s)
    {
        start = s;
        su::GetLoopSetData<int>(set, start, vec, 10, condition_func);
        std::cout << std::endl;
        for (int i = 0; i < vec.size(); ++i)
        {
            std::cout<<vec[i]<<std::endl;
        }
        vec.clear();
    }
    vec.clear();
    std::cout <<"111111111111111111111111111111111111"<<std::endl;
    std::vector<int> vec_con;
    vec_con.push_back(1);
    vec_con.push_back(2);
    vec_con.push_back(3);
    vec_con.push_back(4);
    vec_con.push_back(5);

    unsigned int subpt = 0;
    for (int s = 0; s < 5; ++s)
    {
        subpt = s;
        su::GetLoopVectorData<int>(vec_con, subpt, vec, 4, condition_func);
        std::cout << std::endl;
        for (int i = 0; i < vec.size(); ++i)
        {
            std::cout<<vec[i]<<std::endl;
        }
        vec.clear();
    }
    return 0;
}

int main(int argc, char** argv)
{
    const std::string& str = "7384738";

    std::cout << "秒 " << su::second_time()<<std::endl;
    std::cout << "毫秒 " << su::milli_time()<<std::endl;
    std::cout << "微秒 " << su::micro_time()<<std::endl;
    std::cout << "纳秒 " << su::nano_time()<<std::endl;

    unsigned long int mt = su::micro_time();
    std::cout << "毫秒 " << mt << " 强制转化 "<< (unsigned int)mt <<std::endl;
    std::cout << "当前零点 " << su::zero_sec_time()<<std::endl;
    
    std::cout << "时间戳零点 " << su::zero_sec_time(1614020400)<<std::endl;

    std::cout << "时区 "<< timezone <<std::endl;

    su::get_time_zone();

    std::vector<int> vec;
    su::FloydSample(10, 10, vec);
    for (int i = 0; i< vec.size(); ++i)
    {
        std::cout <<vec[i]<<" ";
    }
    std::cout<<std::endl;
    // vec.clear();
    // std::cout <<" capacity="<<vec.capacity()<<std::endl;
    // for (int i = 0; i< vec.capacity(); ++i)
    // {
    //     std::cout <<vec[i]<<" ";
    // }
    su::DisorderArray(vec);
    for (int i = 0; i< vec.size(); ++i)
    {
        std::cout <<vec[i]<<" ";
    }
    std::cout<<std::endl;    
    su::DisorderArrayV2(vec);
    for (int i = 0; i< vec.size(); ++i)
    {
        std::cout <<vec[i]<<" ";
    }
    std::cout<<std::endl;

    std::cout<<"===================================="<<std::endl;
    std::string s = "3748uu478";
    double n1 = 36384783.749;
    int n = su::string_to_number<int>(s);
    std::string s1 = su::number_to_string(n1);
    std::cout<<" n="<<n<< "    s1 = "<<s1<<std::endl;

    std::string s2 = "2.34";
    float n2 = su::string_to_number<float>(s2);
    std::cout <<" n2="<< n2 <<std::endl;

    std::string s3 = "367:-4738:-4849";
    std::vector<std::string> vec1;
    su::split(s3, ":-", vec1);
    for (int i = 0; i< vec1.size(); ++i)
    {
        std::cout <<vec1[i]<<" ";
    }
    std::cout<<std::endl;
    return 0;
}