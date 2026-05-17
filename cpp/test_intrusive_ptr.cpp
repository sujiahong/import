#include "base/intrusive_release_ptr.h"
#include <iostream>

// 测试类，继承自 BaseRef
class TestClass : public su::BaseRef {
private:
    int value_;
public:
    TestClass(int value = 0) : value_(value) {
        std::cout << "TestClass constructor, value: " << value_ << std::endl;
    }
    
    ~TestClass() {
        std::cout << "TestClass destructor, value: " << value_ << std::endl;
    }
    
    void setValue(int value) {
        value_ = value;
    }
    
    int getValue() const {
        return value_;
    }
};

// 派生测试类，用于测试类型转换
class DerivedTestClass : public TestClass {
private:
    std::string name_;
public:
    DerivedTestClass(int value, const std::string& name) : TestClass(value), name_(name) {
        std::cout << "DerivedTestClass constructor, value: " << value << ", name: " << name_ << std::endl;
    }
    
    ~DerivedTestClass() {
        std::cout << "DerivedTestClass destructor, name: " << name_ << std::endl;
    }
    
    const std::string& getName() const {
        return name_;
    }
};

int main() {
    std::cout << "=== 测试侵入式智能指针 ===" << std::endl;
    
    // 测试 1: 默认构造函数
    std::cout << "\n1. 测试默认构造函数" << std::endl;
    su::IntrusivePtr<TestClass> ptr1;
    std::cout << "ptr1 为空: " << (!ptr1) << std::endl;
    std::cout << "ptr1 引用计数: " << ptr1.use_count() << std::endl;
    
    // 测试 2: 指针构造函数
    std::cout << "\n2. 测试指针构造函数" << std::endl;
    TestClass* rawPtr = new TestClass(10);
    su::IntrusivePtr<TestClass> ptr2(rawPtr);
    std::cout << "ptr2 引用计数: " << ptr2.use_count() << std::endl;
    
    // 测试 3: 拷贝构造函数
    std::cout << "\n3. 测试拷贝构造函数" << std::endl;
    su::IntrusivePtr<TestClass> ptr3(ptr2);
    std::cout << "ptr2 引用计数: " << ptr2.use_count() << std::endl;
    std::cout << "ptr3 引用计数: " << ptr3.use_count() << std::endl;
    
    // 测试 4: 拷贝赋值运算符
    std::cout << "\n4. 测试拷贝赋值运算符" << std::endl;
    su::IntrusivePtr<TestClass> ptr4;
    ptr4 = ptr2;
    std::cout << "ptr2 引用计数: " << ptr2.use_count() << std::endl;
    std::cout << "ptr4 引用计数: " << ptr4.use_count() << std::endl;
    
    // 测试 5: 指针赋值运算符
    std::cout << "\n5. 测试指针赋值运算符" << std::endl;
    TestClass* rawPtr2 = new TestClass(20);
    su::IntrusivePtr<TestClass> ptr5;
    ptr5 = rawPtr2;
    std::cout << "ptr5 引用计数: " << ptr5.use_count() << std::endl;
    
    // 测试 6: 移动构造函数
    std::cout << "\n6. 测试移动构造函数" << std::endl;
    su::IntrusivePtr<TestClass> ptr6(std::move(ptr5));
    std::cout << "ptr5 为空: " << (!ptr5) << std::endl;
    std::cout << "ptr6 引用计数: " << ptr6.use_count() << std::endl;
    
    // 测试 7: 移动赋值运算符
    std::cout << "\n7. 测试移动赋值运算符" << std::endl;
    su::IntrusivePtr<TestClass> ptr7;
    ptr7 = std::move(ptr6);
    std::cout << "ptr6 为空: " << (!ptr6) << std::endl;
    std::cout << "ptr7 引用计数: " << ptr7.use_count() << std::endl;
    
    // 测试 8: 指针运算符
    std::cout << "\n8. 测试指针运算符" << std::endl;
    std::cout << "ptr2->getValue(): " << ptr2->getValue() << std::endl;
    std::cout << "(*ptr2).getValue(): " << (*ptr2).getValue() << std::endl;
    ptr2->setValue(100);
    std::cout << "修改后 ptr2->getValue(): " << ptr2->getValue() << std::endl;
    std::cout << "修改后 ptr3->getValue(): " << ptr3->getValue() << std::endl;
    
    // 测试 9: 类型转换
    std::cout << "\n9. 测试类型转换" << std::endl;
    DerivedTestClass* derivedPtr = new DerivedTestClass(30, "test");
    su::IntrusivePtr<DerivedTestClass> derivedIntrusivePtr(derivedPtr);
    su::IntrusivePtr<TestClass> baseIntrusivePtr = derivedIntrusivePtr;
    std::cout << "baseIntrusivePtr 引用计数: " << baseIntrusivePtr.use_count() << std::endl;
    
    // 测试 convert_to 方法
    DerivedTestClass* convertedPtr = baseIntrusivePtr.convert_to<DerivedTestClass>();
    if (convertedPtr) {
        std::cout << "转换成功，name: " << convertedPtr->getName() << std::endl;
    } else {
        std::cout << "转换失败" << std::endl;
    }
    
    // 测试 10: 比较运算符
    std::cout << "\n10. 测试比较运算符" << std::endl;
    su::IntrusivePtr<TestClass> ptr8 = ptr2;
    su::IntrusivePtr<TestClass> ptr9(new TestClass(40));
    std::cout << "ptr2 == ptr8: " << (ptr2 == ptr8) << std::endl;
    std::cout << "ptr2 != ptr9: " << (ptr2 != ptr9) << std::endl;
    std::cout << "ptr2 < ptr9: " << (ptr2 < ptr9) << std::endl;
    
    // 测试 11: swap 方法
    std::cout << "\n11. 测试 swap 方法" << std::endl;
    std::cout << "交换前 ptr2->getValue(): " << ptr2->getValue() << std::endl;
    std::cout << "交换前 ptr9->getValue(): " << ptr9->getValue() << std::endl;
    ptr2.swap(ptr9);
    std::cout << "交换后 ptr2->getValue(): " << ptr2->getValue() << std::endl;
    std::cout << "交换后 ptr9->getValue(): " << ptr9->getValue() << std::endl;
    
    // 测试 12: make_intrusive 辅助函数
    std::cout << "\n12. 测试 make_intrusive 辅助函数" << std::endl;
    auto ptr10 = su::make_intrusive<TestClass>(50);
    std::cout << "ptr10->getValue(): " << ptr10->getValue() << std::endl;
    std::cout << "ptr10 引用计数: " << ptr10.use_count() << std::endl;
    
    // 测试 13: 引用计数为 0 时自动析构
    std::cout << "\n13. 测试引用计数为 0 时自动析构" << std::endl;
    {
        su::IntrusivePtr<TestClass> tempPtr(new TestClass(60));
        std::cout << "tempPtr 引用计数: " << tempPtr.use_count() << std::endl;
    } // tempPtr 超出作用域，引用计数变为 0，应该自动析构
    
    std::cout << "\n=== 测试完成 ===" << std::endl;
    return 0;
}