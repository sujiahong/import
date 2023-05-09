//#include "./toolbox/original_dependence.hpp"
#include <set>
#include <vector>
#include <iostream>
#include <unistd.h>
#include "./toolbox/util.hpp"
#include "./toolbox/string_function.hpp"

// class TestSig:public su::OnlyOneInstance<TestSig>
// {
// public:
//     TestSig(){}
//     ~TestSig(){}
// };

int main(int argc, char** argv)
{
    // TestSig& t = TestSig::Instance();
    // TestSig t1 = t;
    // TestSig t2(t);
    std::set<int> s;
    s.insert(4);
    s.insert(5);
    s.insert(6);
    s.insert(7);
    s.insert(8);
    std::set<int>::iterator it = s.lower_bound(3);
    if (it == s.end())
        std::cout <<" last迭代器"<<std::endl;
    else
        std::cout <<" it1="<<*it<<std::endl;
    it = s.lower_bound(5);
    if (it == s.end())
        std::cout <<" last迭代器"<<std::endl;
    else
        std::cout <<" it2="<<*it<<std::endl;
    it = s.lower_bound(9);
    if (it == s.end())
        std::cout <<" last迭代器"<<std::endl;
    else
        std::cout <<" it3="<<*it<<std::endl;
    
    double d = su::sqrt_v2(2345.0);
    std::cout<<" 平方根="<<d<<" math:"<<sqrt(2345.0)<<std::endl;

    double d2 = 4.6 * 100;
    std::cout<<" d2="<<(long)d2<<std::endl;
    std::cout<<" su d2="<<su::Double2Int64(d2)<<std::endl;
    std::string s1 = "er,43,65,93";
    std::vector<std::string> vec1;
    su::Split(s1, ",", vec1);
    for (unsigned int i = 0; i < vec1.size(); ++i)
    {
        std::cout<< " i="<<i<<" v="<<vec1[i]<<std::endl;
    }
    std::string s2 = "36334:4443,46:43,54563:65";
    std::map<int, unsigned int> map1;
    su::SplitToMap(s2, ",", ":", map1);
    for (std::map<int, unsigned int>::iterator it = map1.begin(); it != map1.end(); ++it)
    {
        std::cout<< " k="<<it->first<<" v="<<it->second<<std::endl;
    }
    return 0;
}