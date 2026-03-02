////////侵入式智能指针测试

#include "intrusive_release_ptr.h"
#include <iostream>
#include <cassert>
#include <vector>
#include <set>

using namespace su;

// 测试用的基础类
class TestClass : public BaseRef {
public:
    int value;
    static int destruct_count;
    
    TestClass(int v = 0) : value(v) {
        std::cout << "TestClass constructed: " << value << std::endl;
    }
    
    virtual ~TestClass() {
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
        IntrusivePtr<TestClass> ptr1;
        assert(!ptr1);
        assert(ptr1.get() == NULL);
        std::cout << "默认构造函数测试通过" << std::endl;
        
        IntrusivePtr<TestClass> ptr2(new TestClass(10));
        assert(ptr2);
        assert(ptr2->value == 10);
        std::cout << "指针构造函数测试通过" << std::endl;
        
        std::cout << "\n--- 测试拷贝构造 ---" << std::endl;
        IntrusivePtr<TestClass> ptr3 = ptr2;
        assert(ptr3->value == 10);
        assert(ptr2.use_count() == 2);
        assert(ptr3.use_count() == 2);
        std::cout << "拷贝构造函数测试通过，引用计数: " << ptr2.use_count() << std::endl;
        
        std::cout << "\n--- 测试赋值操作 ---" << std::endl;
        IntrusivePtr<TestClass> ptr4;
        ptr4 = ptr2;
        assert(ptr4->value == 10);
        assert(ptr2.use_count() == 3);
        std::cout << "赋值操作测试通过，引用计数: " << ptr2.use_count() << std::endl;
        
        std::cout << "\n--- 测试原始指针赋值 ---" << std::endl;
        IntrusivePtr<TestClass> ptr5;
        ptr5 = new TestClass(20);
        assert(ptr5->value == 20);
        std::cout << "原始指针赋值测试通过" << std::endl;
    }
    
    std::cout << "\n基本功能测试完成" << std::endl;
}

// 测试移动语义
void test_move_semantics() {
    std::cout << "\n========== 测试移动语义 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试移动构造 ---" << std::endl;
        IntrusivePtr<TestClass> ptr1(new TestClass(30));
        assert(ptr1.use_count() == 1);
        
        IntrusivePtr<TestClass> ptr2 = std::move(ptr1);
        assert(ptr2->value == 30);
        assert(ptr2.use_count() == 1);
        assert(!ptr1);  // ptr1 应该为空
        std::cout << "移动构造函数测试通过" << std::endl;
        
        std::cout << "\n--- 测试移动赋值 ---" << std::endl;
        IntrusivePtr<TestClass> ptr3(new TestClass(40));
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
        IntrusivePtr<TestClass> ptr1;
        IntrusivePtr<TestClass> ptr2;
        assert(ptr1 == ptr2);  // 两个空指针应该相等
        std::cout << "空指针相等性测试通过" << std::endl;
        
        IntrusivePtr<TestClass> ptr3(new TestClass(50));
        IntrusivePtr<TestClass> ptr4 = ptr3;
        assert(ptr3 == ptr4);  // 指向同一个对象应该相等
        std::cout << "相同对象相等性测试通过" << std::endl;
        
        IntrusivePtr<TestClass> ptr5(new TestClass(60));
        assert(ptr3 != ptr5);  // 指向不同对象应该不相等
        std::cout << "不同对象不相等性测试通过" << std::endl;
        
        std::cout << "\n--- 测试与原始指针比较 ---" << std::endl;
        TestClass* raw_ptr = new TestClass(70);
        IntrusivePtr<TestClass> ptr6(raw_ptr);
        assert(ptr6 == raw_ptr);
        // 注意：由于显式类型转换，反向比较需要使用 get()
        assert(raw_ptr == ptr6.get());
        std::cout << "与原始指针比较测试通过" << std::endl;
        
        std::cout << "\n--- 测试大小比较 ---" << std::endl;
        IntrusivePtr<TestClass> ptr7(new TestClass(80));
        IntrusivePtr<TestClass> ptr8(new TestClass(90));
        // 比较的是指针地址，不是值
        bool less_than = (ptr7 < ptr8) || (ptr8 < ptr7) || (ptr7 == ptr8);
        assert(less_than);
        std::cout << "大小比较测试通过" << std::endl;
        
        // 测试 <= 和 >=
        assert(ptr7 <= ptr7);
        assert(ptr7 >= ptr7);
        std::cout << "<= 和 >= 测试通过" << std::endl;
    }
    
    std::cout << "\n比较运算符测试完成" << std::endl;
}

// 测试类型转换
void test_type_conversion() {
    std::cout << "\n========== 测试类型转换 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试 dynamic_cast ---" << std::endl;
        IntrusivePtr<TestClass> base_ptr(new DerivedClass(100, "TestDerived"));
        
        DerivedClass* derived = base_ptr.convert_to<DerivedClass>();
        assert(derived != NULL);
        assert(derived->value == 100);
        assert(derived->name == "TestDerived");
        derived->doDerived();
        std::cout << "dynamic_cast 测试通过" << std::endl;
        
        std::cout << "\n--- 测试失败的类型转换 ---" << std::endl;
        IntrusivePtr<TestClass> base_ptr2(new TestClass(110));
        DerivedClass* derived2 = base_ptr2.convert_to<DerivedClass>();
        assert(derived2 == NULL);
        std::cout << "失败的类型转换测试通过" << std::endl;
        
        std::cout << "\n--- 测试显式类型转换 ---" << std::endl;
        IntrusivePtr<TestClass> ptr(new TestClass(120));
        bool exists = static_cast<bool>(ptr);
        assert(exists);
        std::cout << "显式 bool 转换测试通过" << std::endl;
        
        // 注意：由于现在是显式转换，以下代码需要显式转换
        TestClass* raw = static_cast<TestClass*>(ptr);
        assert(raw->value == 120);
        std::cout << "显式指针转换测试通过" << std::endl;
    }
    
    std::cout << "\n类型转换测试完成" << std::endl;
}

// 测试 swap 功能
void test_swap() {
    std::cout << "\n========== 测试 swap 功能 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试成员 swap ---" << std::endl;
        IntrusivePtr<TestClass> ptr1(new TestClass(130));
        IntrusivePtr<TestClass> ptr2(new TestClass(140));
        
        assert(ptr1->value == 130);
        assert(ptr2->value == 140);
        
        ptr1.swap(ptr2);
        
        assert(ptr1->value == 140);
        assert(ptr2->value == 130);
        std::cout << "成员 swap 测试通过" << std::endl;
        
        std::cout << "\n--- 测试全局 swap ---" << std::endl;
        swap(ptr1, ptr2);
        assert(ptr1->value == 130);
        assert(ptr2->value == 140);
        std::cout << "全局 swap 测试通过" << std::endl;
    }
    
    std::cout << "\nswap 功能测试完成" << std::endl;
}

// 测试在容器中的使用
void test_in_containers() {
    std::cout << "\n========== 测试在容器中的使用 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试在 vector 中使用 ---" << std::endl;
        std::vector<IntrusivePtr<TestClass>> vec;
        vec.push_back(IntrusivePtr<TestClass>(new TestClass(150)));
        vec.push_back(IntrusivePtr<TestClass>(new TestClass(160)));
        vec.push_back(IntrusivePtr<TestClass>(new TestClass(170)));
        
        assert(vec[0]->value == 150);
        assert(vec[1]->value == 160);
        assert(vec[2]->value == 170);
        std::cout << "vector 测试通过" << std::endl;
        
        std::cout << "\n--- 测试在 set 中使用 ---" << std::endl;
        std::set<IntrusivePtr<TestClass>> ptr_set;
        ptr_set.insert(IntrusivePtr<TestClass>(new TestClass(180)));
        ptr_set.insert(IntrusivePtr<TestClass>(new TestClass(190)));
        ptr_set.insert(IntrusivePtr<TestClass>(new TestClass(200)));
        
        assert(ptr_set.size() == 3);
        std::cout << "set 测试通过" << std::endl;
    }
    
    std::cout << "\n容器测试完成" << std::endl;
}

// 测试 make_intrusive 辅助函数
void test_make_intrusive() {
    std::cout << "\n========== 测试 make_intrusive ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试 make_intrusive ---" << std::endl;
        auto ptr = make_intrusive<TestClass>(210);
        assert(ptr->value == 210);
        assert(ptr.use_count() == 1);
        std::cout << "make_intrusive 测试通过" << std::endl;
        
        std::cout << "\n--- 测试 make_intrusive 带多个参数 ---" << std::endl;
        auto derived_ptr = make_intrusive<DerivedClass>(220, "TestName");
        assert(derived_ptr->value == 220);
        assert(derived_ptr->name == "TestName");
        std::cout << "make_intrusive 多参数测试通过" << std::endl;
    }
    
    std::cout << "\nmake_intrusive 测试完成" << std::endl;
}

// 测试逻辑非操作符
void test_logical_not() {
    std::cout << "\n========== 测试逻辑非操作符 ==========" << std::endl;
    
    {
        std::cout << "\n--- 测试逻辑非 ---" << std::endl;
        IntrusivePtr<TestClass> ptr1;
        assert(!ptr1);  // 空指针应该返回 true
        std::cout << "空指针逻辑非测试通过" << std::endl;
        
        IntrusivePtr<TestClass> ptr2(new TestClass(230));
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
        IntrusivePtr<TestClass> ptr(new TestClass(240));
        ptr->doSomething();
        assert(ptr->value == 240);
        std::cout << "箭头操作符测试通过" << std::endl;
        
        std::cout << "\n--- 测试解引用操作符 ---" << std::endl;
        (*ptr).doSomething();
        assert((*ptr).value == 240);
        std::cout << "解引用操作符测试通过" << std::endl;
    }
    
    std::cout << "\n箭头和解引用操作符测试完成" << std::endl;
}

// 主函数
int main() {
    std::cout << "======================================" << std::endl;
    std::cout << "    侵入式智能指针接口测试开始" << std::endl;
    std::cout << "======================================" << std::endl;
    
    test_basic_functionality();
    test_move_semantics();
    test_comparison_operators();
    test_type_conversion();
    test_swap();
    test_in_containers();
    test_make_intrusive();
    test_logical_not();
    test_arrow_and_dereference();
    
    std::cout << "\n======================================" << std::endl;
    std::cout << "    所有测试完成！" << std::endl;
    std::cout << "    析构的对象数: " << TestClass::destruct_count << std::endl;
    std::cout << "======================================" << std::endl;
    
    return 0;
}
