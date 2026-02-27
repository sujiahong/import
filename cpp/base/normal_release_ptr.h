////////普通非侵入式智能指针

#ifndef NORMAL_RELEASE_PTR_H
#define NORMAL_RELEASE_PTR_H

#include <iostream>

namespace su {

// 引用计数类
template<typename T>
class RefCount {
private:
    T* ptr;          // 指向实际对象的指针
    int ref_count;    // 引用计数

public:
    // 构造函数
    RefCount(T* p = nullptr) : ptr(p), ref_count(1) {}

    // 增加引用计数
    void add_ref() {
        ref_count++;
    }

    // 减少引用计数并检查是否需要释放
    bool release() {
        ref_count--;
        return ref_count == 0;
    }

    // 获取引用计数
    int get_count() const {
        return ref_count;
    }

    // 获取原始指针
    T* get() const {
        return ptr;
    }

    // 析构函数
    ~RefCount() {
        delete ptr;
    }
};

// 非侵入式智能指针
template<typename T>
class NormalReleasePtr {
private:
    RefCount<T>* ref_count;  // 指向引用计数对象的指针

public:
    // 构造函数
    NormalReleasePtr(T* p = nullptr) {
        if (p) {
            ref_count = new RefCount<T>(p);
        } else {
            ref_count = nullptr;
        }
    }

    // 拷贝构造函数
    NormalReleasePtr(const NormalReleasePtr<T>& other) {
        if (other.ref_count) {
            ref_count = other.ref_count;
            ref_count->add_ref();
        } else {
            ref_count = nullptr;
        }
    }

    // 赋值操作符
    NormalReleasePtr<T>& operator=(const NormalReleasePtr<T>& other) {
        if (this != &other) {
            // 先释放当前资源
            if (ref_count && ref_count->release()) {
                delete ref_count;
            }
            
            // 复制新资源
            if (other.ref_count) {
                ref_count = other.ref_count;
                ref_count->add_ref();
            } else {
                ref_count = nullptr;
            }
        }
        return *this;
    }

    // 移动构造函数
    NormalReleasePtr(NormalReleasePtr<T>&& other) noexcept {
        ref_count = other.ref_count;
        other.ref_count = nullptr;
    }

    // 移动赋值操作符
    NormalReleasePtr<T>& operator=(NormalReleasePtr<T>&& other) noexcept {
        if (this != &other) {
            // 先释放当前资源
            if (ref_count && ref_count->release()) {
                delete ref_count;
            }
            
            // 移动资源
            ref_count = other.ref_count;
            other.ref_count = nullptr;
        }
        return *this;
    }

    // 析构函数
    ~NormalReleasePtr() {
        if (ref_count && ref_count->release()) {
            delete ref_count;
        }
    }

    // 解引用操作符
    T& operator*() const {
        return *(ref_count->get());
    }

    // 箭头操作符
    T* operator->() const {
        return ref_count->get();
    }

    // 获取原始指针
    T* get() const {
        return ref_count ? ref_count->get() : nullptr;
    }

    // 检查是否为空
    bool is_null() const {
        return ref_count == nullptr || ref_count->get() == nullptr;
    }

    // 重载bool操作符
    explicit operator bool() const {
        return !is_null();
    }

    // 交换两个智能指针
    void swap(NormalReleasePtr<T>& other) {
        std::swap(ref_count, other.ref_count);
    }

    // 获取引用计数
    int use_count() const {
        return ref_count ? ref_count->get_count() : 0;
    }
};

// 辅助函数：创建智能指针
template<typename T, typename... Args>
NormalReleasePtr<T> make_normal_release_ptr(Args&&... args) {
    return NormalReleasePtr<T>(new T(std::forward<Args>(args)...));
}

// 辅助函数：交换两个智能指针
template<typename T>
void swap(NormalReleasePtr<T>& lhs, NormalReleasePtr<T>& rhs) {
    lhs.swap(rhs);
}

// 重载比较操作符
template<typename T>
bool operator==(const NormalReleasePtr<T>& lhs, const NormalReleasePtr<T>& rhs) {
    return lhs.get() == rhs.get();
}

template<typename T>
bool operator!=(const NormalReleasePtr<T>& lhs, const NormalReleasePtr<T>& rhs) {
    return lhs.get() != rhs.get();
}

template<typename T>
bool operator<(const NormalReleasePtr<T>& lhs, const NormalReleasePtr<T>& rhs) {
    return lhs.get() < rhs.get();
}

template<typename T>
bool operator<=(const NormalReleasePtr<T>& lhs, const NormalReleasePtr<T>& rhs) {
    return lhs.get() <= rhs.get();
}

template<typename T>
bool operator>(const NormalReleasePtr<T>& lhs, const NormalReleasePtr<T>& rhs) {
    return lhs.get() > rhs.get();
}

template<typename T>
bool operator>=(const NormalReleasePtr<T>& lhs, const NormalReleasePtr<T>& rhs) {
    return lhs.get() >= rhs.get();
}

} // namespace base

#endif // NORMAL_RELEASE_PTR_H