////////侵入式智能指针

#ifndef __INTROUSIVE_PTR_H__
#define __INTROUSIVE_PTR_H__

namespace su {
class BaseRef
{
private:
    int refence_count_ = 0;////引用计数
public:
    inline BaseRef(): refence_count_(0)
    {}

    virtual ~BaseRef()
    {}
    inline int add_ref()
    {
        return __sync_add_and_fetch(&refence_count_, 1);
    }
    inline int reduce_ref()
    {
        int ret = __sync_sub_and_fetch(&refence_count_, 1);
        if (ret != 0)
            return ret;
        delete this; //////析构自己
        return 0;
    }
    
    // 获取引用计数
    inline int get_ref_count() const
    {
        return refence_count_;
    }
};

/// @brief 侵入式智能指针   管理原始指针 
template <class T>
class IntrusivePtr 
{
public:
    typedef IntrusivePtr<T> SelfType;
    typedef T ElementType;
private:
    ElementType* ptr_;
public:
    inline IntrusivePtr():ptr_(NULL)
    {}
    ~IntrusivePtr()
    {
        if (ptr_)
        {
            ptr_->reduce_ref();
            ptr_ = NULL;
        }
    }
    inline IntrusivePtr(const IntrusivePtr& right)
    {
        ptr_ = right.ptr_;
        if (ptr_)
        {
            ptr_->add_ref();
        }
    }
    inline IntrusivePtr(ElementType* tmp_ptr)
    {
        ptr_ = tmp_ptr;
        if (ptr_)
        {
            ptr_->add_ref();
        }
    }
    inline IntrusivePtr& operator=(const IntrusivePtr& right)
    {
        if (ptr_)
        {
            ptr_->reduce_ref();
        }
        ptr_ = right.ptr_;
        if (ptr_)
        {
            ptr_->add_ref();
        }
        return *this;
    }
    inline IntrusivePtr& operator= (ElementType* tmp_ptr)
    {
        if (ptr_)
        {
            ptr_->reduce_ref();
        }
        ptr_ = tmp_ptr;
        if (ptr_)
        {
            ptr_->add_ref();
        }
        return *this;
    }
    
    // 移动构造函数
    inline IntrusivePtr(IntrusivePtr&& right) noexcept
    {
        ptr_ = right.ptr_;
        right.ptr_ = NULL;
    }
    
    // 移动赋值运算符
    inline IntrusivePtr& operator=(IntrusivePtr&& right) noexcept
    {
        if (this != &right)
        {
            if (ptr_)
            {
                ptr_->reduce_ref();
            }
            ptr_ = right.ptr_;
            right.ptr_ = NULL;
        }
        return *this;
    }
    template <typename Q>
    inline const Q* convert_to() const
    {
        if (ptr_)
            return dynamic_cast<const Q*>(ptr_);
        else
            return NULL;
    }
    
    template <typename Q>
    inline Q* convert_to()
    {
        if (ptr_)
            return dynamic_cast<Q*>(ptr_);
        else
            return NULL;
    }
    ///////指针运算相关
    inline bool operator== (const IntrusivePtr& right) const
    {
        return (ptr_ == right.ptr_);
    }
    inline bool operator!= (const IntrusivePtr& right) const
    {
        return (ptr_ != right.ptr_);
    }
    inline bool operator< (const IntrusivePtr& right) const
    {
        return (ptr_ < right.ptr_);
    }
    inline bool operator== (ElementType* tmp_ptr) const 
    {
        return (ptr_ == tmp_ptr);
    }
    inline bool operator!= (ElementType* tmp_ptr) const
    {
        return (ptr_ != tmp_ptr);
    }
    inline bool operator< (ElementType* tmp_ptr) const
    {
        return (ptr_ < tmp_ptr);
    }
    inline bool operator== (const ElementType* tmp_ptr) const 
    {
        return (ptr_ == tmp_ptr);
    }
    inline bool operator!= (const ElementType* tmp_ptr) const
    {
        return (ptr_ != tmp_ptr);
    }
    inline bool operator< (const ElementType* tmp_ptr) const
    {
        return (ptr_ < tmp_ptr);
    }
    // 添加其他比较运算符
    inline bool operator<= (const IntrusivePtr& right) const
    {
        return (ptr_ <= right.ptr_);
    }
    inline bool operator> (const IntrusivePtr& right) const
    {
        return (ptr_ > right.ptr_);
    }
    inline bool operator>= (const IntrusivePtr& right) const
    {
        return (ptr_ >= right.ptr_);
    }

    inline bool operator! () const
    {
        return (ptr_ == NULL);
    }
    inline ElementType* operator-> () const
    {
        return ptr_;
    }
    inline ElementType& operator* () const 
    {
        return *ptr_;
    }
    explicit inline operator bool () const ///类型转换
    {
        return (ptr_ != NULL);
    }
    explicit inline operator ElementType& () const ////类型转换
    {
        return *ptr_;
    }
    explicit inline operator ElementType* () const ////类型转换
    {
        return ptr_;
    }
    inline ElementType* get() const
    {
        return ptr_;
    }
    
    // 交换两个智能指针
    inline void swap(IntrusivePtr& other)
    {
        ElementType* temp = ptr_;
        ptr_ = other.ptr_;
        other.ptr_ = temp;
    }
    
    // 获取引用计数
    inline int use_count() const
    {
        if (ptr_)
        {
            return ptr_->get_ref_count();
        }
        return 0;
    }
};

// 全局swap辅助函数
template <class T>
inline void swap(IntrusivePtr<T>& lhs, IntrusivePtr<T>& rhs)
{
    lhs.swap(rhs);
}

// 辅助函数：创建侵入式智能指针
template <class T, typename... Args>
inline IntrusivePtr<T> make_intrusive(Args&&... args)
{
    return IntrusivePtr<T>(new T(std::forward<Args>(args)...));
}

}
#endif