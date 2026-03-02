////////非侵入式智能指针测试

#include "normal_release_ptr.h"
#include <iostream>
#include <cassert>
#include <vector>
#include <set>

using namespace su;

// 测试用的类（不需要继承任何基类）
class TestClass {
public:
    int value;
    static int destruct_count;
    
    TestClass(int v = 0) : value(v) {
        std::cout << "TestClass constructed: " << value << std::endl;
    }
    
    ~TestClass() {
        std::cout << "TestClass destructed: " << value << std::endl;
        destruct_count++;
    }
    
    void doSomething() {
        std::cout << "Doing something with value: " << value << std::endl;
    }
};

int TestClass::destruct_count = 0;

// 派生类，用于测试类型转换
class DerivedClass : public TestClass {
public:
    std::string name;
    
    DerivedClass(int v, const std::string& n) : TestClass(v), name(n) {
        std::cout << "DerivedClass constructed: " << name << std::endl;
    }
    
    ~DerivedClass() {
        std::cout << "DerivedClass destructed: " << name << std::endl;
    }
    
    void doDerived() {
        std::cout << "Derived class doing something: " << name << std::endl;
    }
};

// 测试基本功能
void test_basic_functionality() {
    std::cout << "\n========== 测试基本功能 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试构造函数 ---" << std::endl;
        NormalReleasePtr<TestClass> ptr1;
        assert(!ptr1);
        assert(ptr1.get() == nullptr);
        std::cout << "默认构造函数测试通过" << std::endl;
        
        NormalReleasePtr<TestClass> ptr2(new TestClass(10));
        assert(ptr2);
        assert(ptr2->value == 10);
        std::cout << "指针构造函数测试通过" << std::endl;
        
        std::cout << "\n--- 测试拷贝构造 ---" << std::endl;
        NormalReleasePtr<TestClass> ptr3 = ptr2;
        assert(ptr3->value == 10);
        assert(ptr2.use_count() == 2);
        assert(ptr3.use_count() == 2);
        std::cout << "拷贝构造函数测试通过，引用计数: " << ptr2.use_count() << std::endl;
        
        std::cout << "\n--- 测试赋值操作 ---" << std::endl;
        NormalReleasePtr<TestClass> ptr4;
        ptr4 = ptr2;
        assert(ptr4->value == 10);
        assert(ptr2.use_count() == 3);
        std::cout << "赋值操作测试通过，引用计数: " << ptr2.use_count() << std::endl;
    }
    
    std::cout << "\n基本功能测试完成" << std::endl;
}

// 测试移动语义
void test_move_semantics() {
    std::cout << "\n========== 测试移动语义 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试移动构造 ---" << std::endl;
        NormalReleasePtr<TestClass> ptr1(new TestClass(30));
        assert(ptr1.use_count() == 1);
        
        NormalReleasePtr<TestClass> ptr2 = std::move(ptr1);
        assert(ptr2->value == 30);
        assert(ptr2.use_count() == 1);
        assert(!ptr1);  // ptr1 应该为空
        std::cout << "移动构造函数测试通过" << std::endl;
        
        std::cout << "\n--- 测试移动赋值 ---" << std::endl;
        NormalReleasePtr<TestClass> ptr3(new TestClass(40));
        ptr3 = std::move(ptr2);
        assert(ptr3->value == 30);
        assert(!ptr2);  // ptr2 应该为空
        std::cout << "移动赋值操作测试通过" << std::endl;
    }
    
    std::cout << "\n移动语义测试完成" << std::endl;
}

// 测试比较运算符
void test_comparison_operators() {
    std::cout << "\n========== 测试比较运算符 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试相等性比较 ---" << std::endl;
        NormalReleasePtr<TestClass> ptr1;
        NormalReleasePtr<TestClass> ptr2;
        assert(ptr1 == ptr2);  // 两个空指针应该相等
        std::cout << "空指针相等性测试通过" << std::endl;
        
        NormalReleasePtr<TestClass> ptr3(new TestClass(50));
        NormalReleasePtr<TestClass> ptr4 = ptr3;
        assert(ptr3 == ptr4);  // 指向同一个对象应该相等
        std::cout << "相同对象相等性测试通过" << std::endl;
        
        NormalReleasePtr<TestClass> ptr5(new TestClass(60));
        assert(ptr3 != ptr5);  // 指向不同对象应该不相等
        std::cout << "不同对象不相等性测试通过" << std::endl;
        
        std::cout << "\n--- 测试大小比较 ---" << std::endl;
        NormalReleasePtr<TestClass> ptr6(new TestClass(70));
        NormalReleasePtr<TestClass> ptr7(new TestClass(80));
        // 比较的是指针地址，不是值
        bool less_than = (ptr6 < ptr7) || (ptr7 < ptr6) || (ptr6 == ptr7);
        assert(less_than);
        std::cout << "大小比较测试通过" << std::endl;
        
        // 测试 <= 和 >=
        assert(ptr6 <= ptr6);
        assert(ptr6 >= ptr6);
        std::cout << "<= 和 >= 测试通过" << std::endl;
    }
    
    std::cout << "\n比较运算符测试完成" << std::endl;
}

// 测试类型转换
void test_type_conversion() {
    std::cout << "\n========== 测试类型转换 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试显式类型转换 ---" << std::endl;
        NormalReleasePtr<TestClass> ptr(new TestClass(90));
        bool exists = static_cast<bool>(ptr);
        assert(exists);
        std::cout << "显式 bool 转换测试通过" << std::endl;
        
        // 测试派生类
        NormalReleasePtr<DerivedClass> derived_ptr(new DerivedClass(100, "TestDerived"));
        assert(derived_ptr->value == 100);
        assert(derived_ptr->name == "TestDerived");
        std::cout << "派生类测试通过" << std::endl;
    }
    
    std::cout << "\n类型转换测试完成" << std::endl;
}

// 测试 swap 功能
void test_swap() {
    std::cout << "\n========== 测试 swap 功能 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试成员 swap ---" << std::endl;
        NormalReleasePtr<TestClass> ptr1(new TestClass(110));
        NormalReleasePtr<TestClass> ptr2(new TestClass(120));
        
        assert(ptr1->value == 110);
        assert(ptr2->value == 120);
        
        ptr1.swap(ptr2);
        
        assert(ptr1->value == 120);
        assert(ptr2->value == 110);
        std::cout << "成员 swap 测试通过" << std::endl;
        
        std::cout << "\n--- 测试全局 swap ---" << std::endl;
        swap(ptr1, ptr2);
        assert(ptr1->value == 110);
        assert(ptr2->value == 120);
        std::cout << "全局 swap 测试通过" << std::endl;
    }
    
    std::cout << "\nswap 功能测试完成" << std::endl;
}

// 测试在容器中的使用
void test_in_containers() {
    std::cout << "\n========== 测试在容器中的使用 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试在 vector 中使用 ---" << std::endl;
        std::vector<NormalReleasePtr<TestClass>> vec;
        vec.push_back(NormalReleasePtr<TestClass>(new TestClass(130)));
        vec.push_back(NormalReleasePtr<TestClass>(new TestClass(140)));
        vec.push_back(NormalReleasePtr<TestClass>(new TestClass(150)));
        
        assert(vec[0]->value == 130);
        assert(vec[1]->value == 140);
        assert(vec[2]->value == 150);
        std::cout << "vector 测试通过" << std::endl;
        
        std::cout << "\n--- 测试在 set 中使用 ---" << std::endl;
        std::set<NormalReleasePtr<TestClass>> ptr_set;
        ptr_set.insert(NormalReleasePtr<TestClass>(new TestClass(160)));
        ptr_set.insert(NormalReleasePtr<TestClass>(new TestClass(170)));
        ptr_set.insert(NormalReleasePtr<TestClass>(new TestClass(180)));
        
        assert(ptr_set.size() == 3);
        std::cout << "set 测试通过" << std::endl;
    }
    
    std::cout << "\n容器测试完成" << std::endl;
}

// 测试 make_normal_release_ptr 辅助函数
void test_make_normal_release_ptr() {
    std::cout << "\n========== 测试 make_normal_release_ptr ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试 make_normal_release_ptr ---" << std::endl;
        auto ptr = make_normal_release_ptr<TestClass>(190);
        assert(ptr->value == 190);
        assert(ptr.use_count() == 1);
        std::cout << "make_normal_release_ptr 测试通过" << std::endl;
        
        std::cout << "\n--- 测试 make_normal_release_ptr 带多个参数 ---" << std::endl;
        auto derived_ptr = make_normal_release_ptr<DerivedClass>(200, "TestName");
        assert(derived_ptr->value == 200);
        assert(derived_ptr->name == "TestName");
        std::cout << "make_normal_release_ptr 多参数测试通过" << std::endl;
    }
    
    std::cout << "\nmake_normal_release_ptr 测试完成" << std::endl;
}

// 测试逻辑非操作符
void test_logical_not() {
    std::cout << "\n========== 测试逻辑非操作符 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试逻辑非 ---" << std::endl;
        NormalReleasePtr<TestClass> ptr1;
        assert(!ptr1);  // 空指针应该返回 true
        std::cout << "空指针逻辑非测试通过" << std::endl;
        
        NormalReleasePtr<TestClass> ptr2(new TestClass(210));
        assert(!(!ptr2));  // 非空指针应该返回 false
        std::cout << "非空指针逻辑非测试通过" << std::endl;
    }
    
    std::cout << "\n逻辑非操作符测试完成" << std::endl;
}

// 测试箭头和解引用操作符
void test_arrow_and_dereference() {
    std::cout << "\n========== 测试箭头和解引用操作符 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试箭头操作符 ---" << std::endl;
        NormalReleasePtr<TestClass> ptr(new TestClass(220));
        ptr->doSomething();
        assert(ptr->value == 220);
        std::cout << "箭头操作符测试通过" << std::endl;
        
        std::cout << "\n--- 测试解引用操作符 ---" << std::endl;
        (*ptr).doSomething();
        assert((*ptr).value == 220);
        std::cout << "解引用操作符测试通过" << std::endl;
    }
    
    std::cout << "\n箭头和解引用操作符测试完成" << std::endl;
}

// 主函数
int main() {
    std::cout << "======================================" << std::endl;
    std::cout << "    非侵入式智能指针接口测试开始" << std::endl;
    std::cout << "======================================" << std::endl;
    
    test_basic_functionality();
    test_move_semantics();
    test_comparison_operators();
    test_type_conversion();
    test_swap();
    test_in_containers();
    test_make_normal_release_ptr();
    test_logical_not();
    test_arrow_and_dereference();
    
    std::cout << "\n======================================" << std::endl;
    std::cout << "    所有测试完成！" << std::endl;
    std::cout << "    析构的对象数: " << TestClass::destruct_count << std::endl;
    std::cout << "======================================" << std::endl;
    
    return 0;
}
