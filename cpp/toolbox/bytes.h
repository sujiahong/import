
#include <stdlib.h>
#include <string.h>
#include <string>

namespace su 
{
class Bytes
{
private:
    char* data_;
    unsigned int len_;/////不包含‘\0'的长度
    // 私有方法用于分配和初始化内存
    void AllocateAndCopy(const char* n_data, unsigned int n_len)
    {
        data_ = new (std::nothrow) char[n_len + 1];
        if (!data_)
        {
            throw std::bad_alloc();
        }
        len_ = n_len;
        memcpy(data_, n_data, len_);
        data_[len_] = '\0'; // Ensure null-termination
    }
public:
    Bytes()
    {
        data_ = NULL;
        len_ = 0;
    }
    ~Bytes()
    {
        if (data_)
        {
            delete [] data_;
            data_ = NULL;
            len_ = 0;
        }
    }
public:
    inline const char* GetData() const
    {
        return data_;
    }
    inline const unsigned int GetLen() const
    {
        return len_;
    }
    Bytes(unsigned int len)
    {
        // data_ = new char[len+1];
        // len_ = len;
        AllocateAndCopy("", len);
    }
    Bytes(const char* str)
    {
        AllocateAndCopy(str, strlen(str));
    }
    Bytes(const std::string& str)
    {
        AllocateAndCopy(str.c_str(), str.size());
    }
    Bytes(const Bytes& bytes)
    {
        AllocateAndCopy(bytes.GetData(), bytes.GetLen());
    }
    Bytes(const char* str, unsigned int len)
    {
        AllocateAndCopy(str, len);
    }
    //赋值运算符重载只能有一个参数
    inline Bytes& operator=(const Bytes& right)
    {
        if (this != &right)
        {
            ResetBytes(right.GetData(), right.GetLen());
        }
        return *this;
    }
    inline Bytes& operator=(const char* str)
    {
        ResetBytes(str, strlen(str));
        return *this;
    }
    inline Bytes& operator=(const std::string& str)
    {
        ResetBytes(str.c_str(), str.size());
        return *this;
    }

    inline bool operator==(const Bytes& right) const
    {
        if (len_ == right.GetLen())
        {
            return (memcmp(data_, right.GetData(), len_) == 0);
        }
        else
            return false;
    }
private:
    inline void ResetBytes(const char* n_data, unsigned int n_len)
    {
        if (data_)
        {
            if (len_ >= n_len)
            {
                len_ = n_len;
                memcpy(data_, n_data, len_);
                data_[len_] = '\0';
            }
            else
            {
                delete [] data_;
                AllocateAndCopy(n_data, n_len);
            }
        }
        else
        {
            AllocateAndCopy(n_data, n_len);
        }
    }
};
}