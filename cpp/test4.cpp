#include "./toolbox/original_dependence.hpp"
#include <set>
#include <vector>
#include <iostream>
#include <unistd.h>


class TestSig:public su::OnlyOneInstance<TestSig>
{
public:
    TestSig(){}
    ~TestSig(){}
};

int main(int argc, char** argv)
{
    TestSig& t = TestSig::Instance();
    TestSig t1 = t;
    TestSig t2(t);
    return 0;
}