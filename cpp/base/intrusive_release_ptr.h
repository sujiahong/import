////////侵入式智能指针

#ifndef __INTROUSIVE_RELEASE_PTR_H__
#define __INTROUSIVE_RELEASE_PTR_H__ 

namespace su {
class base_release
{
private:
    int refence_count_ = 0;////引用计数
public:
    inline base_release(): refence_count_(0)
    {}

    virtual ~base_release()
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
};

/// @brief 侵入式智能指针   管理原始指针 
template <class T>
class intrusive_ptr 
{
public:
    typedef intrusive_ptr<T> SelfType;
    typedef T ElementType;
private:
    ElementType* ptr_;
public:
    inline intrusive_ptr():ptr_(NULL)
    {}
    virtual ~intrusive_ptr()
    {
        if (ptr_)
        {
            ptr_->reduce_ref();
            ptr_ = NULL;
        }
    }
    inline intrusive_ptr(const intrusive_ptr& right)
    {
        ptr_ = right.ptr_;
        if (ptr_)
        {
            ptr_->add_ref();
        }
    }
    inline intrusive_ptr(ElementType* tmp_ptr)
    {
        ptr_ = tmp_ptr;
        if (ptr_)
        {
            ptr_->add_ref();
        }
    }
    inline intrusive_ptr& operator=(const intrusive_ptr& right)
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
    inline intrusive_ptr& operator= (ElementType* tmp_ptr)
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
    template <typename Q>
    inline const Q* convert_to() const
    {
        if (ptr_)
            return static_cast<const Q*>(ptr_);
        else
            return NULL;
    }
    ///////指针运算相关
    inline bool operator== (const intrusive_ptr& right) const
    {
        if (ptr_ == NULL) return false;
        return (ptr_ == right.ptr_);
    }
    inline bool operator!= (const intrusive_ptr& right) const
    {
        if (ptr_ == NULL) return true;
        return (ptr_ != right.ptr_);

    }
    inline bool operator< (const intrusive_ptr& right) const
    {
        if (ptr_ == NULL) return true;
        return (ptr_ < right.ptr_);
    }
    inline bool operator== (ElementType* tmp_ptr) const 
    {
        if (ptr_ == NULL) return false;
        return (ptr_ == tmp_ptr);
    }
    inline bool operator!= (ElementType* tmp_ptr) const
    {
        if (ptr_ == NULL) return true;
        return (ptr_ != tmp_ptr);
    }
    inline bool operator< (ElementType* tmp_ptr) const
    {
        if (ptr_ == NULL) return true;
        return (ptr_ < tmp_ptr);
    }
    inline bool operator== (const ElementType* tmp_ptr) const 
    {
        if (ptr_ == NULL) return false;
        return (ptr_ == tmp_ptr);
    }
    inline bool operator!= (const ElementType* tmp_ptr) const
    {
        if (ptr_ == NULL) return true;
        return (ptr_ != tmp_ptr);
    }
    inline bool operator< (const ElementType* tmp_ptr) const
    {
        if (ptr_ == NULL) return true;
        return (ptr_ < tmp_ptr);
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
    inline operator bool () const ///类型转换
    {
        return (ptr_ != NULL);
    }
    inline operator ElementType& () const ////类型转换
    {
        return *ptr_;
    }
    inline operator ElementType* () const ////类型转换
    {
        return ptr_;
    }
    inline ElementType* get() const
    {
        return ptr_;
    }
};

}
#endif