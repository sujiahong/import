//#include "./toolbox/original_dependence.hpp"
#include <set>
#include <vector>
#include <iostream>
#include <unistd.h>


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
    // if (it == s.end())
    //     std::cout <<" last迭代器"<<std::endl;
    // else
    //     std::cout <<" it1="<<*it<<std::endl;
    // it = s.lower_bound(5);
    // if (it == s.end())
    //     std::cout <<" last迭代器"<<std::endl;
    // else
    //     std::cout <<" it2="<<*it<<std::endl;
    it = s.lower_bound(9);
    if (it == s.end())
        std::cout <<" last迭代器"<<std::endl;
    else
        std::cout <<" it3="<<*it<<std::endl;
    return 0;
}