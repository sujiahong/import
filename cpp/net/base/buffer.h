#ifndef __BUFFER_H__
#define __BUFFER_H__

#include "../../base/original_base.h"

#include <vector>
#include <string>
#include <cstring>
#include <algorithm>
#include <sys/uio.h>
#include <errno.h>

namespace su
{
/// @brief 网络收发缓冲区（muduo 风格）
/// 布局： [prependable | readable | writable]
///         m_read_idx_   m_write_idx_   size
class Buffer: public Copyable {
public:
    static const size_t kCheapPrepend = 8;   ////头部预留字节，用于日后加包头
    static const size_t kInitialSize = 4096; ////初始可写大小
private:
    std::vector<char> m_buffer_;
    size_t m_read_idx_;
    size_t m_write_idx_;
public:
    Buffer(size_t a_size = kInitialSize)
        : m_buffer_(kCheapPrepend + a_size),
          m_read_idx_(kCheapPrepend),
          m_write_idx_(kCheapPrepend)
    {}

    ~Buffer() {}

    ////可读字节数
    size_t ReadableBytes() const { return m_write_idx_ - m_read_idx_; }
    ////可写字节数
    size_t WritableBytes() const { return m_buffer_.size() - m_write_idx_; }
    ////头部预留字节数
    size_t PrependableBytes() const { return m_read_idx_; }

    ////读指针
    const char* Peek() const { return begin() + m_read_idx_; }
    char* Peek() { return begin() + m_read_idx_; }
    ////写指针
    char* BeginWrite() { return begin() + m_write_idx_; }
    const char* BeginWrite() const { return begin() + m_write_idx_; }

    ////取出 len 字节（读指针后移）
    void Retrieve(size_t len)
    {
        if (len < ReadableBytes())
        {
            m_read_idx_ += len;
        }
        else
        {
            RetrieveAll();
        }
    }
    void RetrieveAll()
    {
        m_read_idx_ = kCheapPrepend;
        m_write_idx_ = kCheapPrepend;
    }
    ////取出并返回字符串
    std::string RetrieveAsString(size_t len)
    {
        std::string result(Peek(), len);
        Retrieve(len);
        return result;
    }
    std::string RetrieveAllAsString()
    {
        return RetrieveAsString(ReadableBytes());
    }

    ////追加数据
    void Append(const char* a_data, size_t a_len)
    {
        EnsureWritable(a_len);
        std::copy(a_data, a_data + a_len, BeginWrite());
        HasWritten(a_len);
    }
    void Append(const std::string& a_str)
    {
        Append(a_str.data(), a_str.size());
    }

    void EnsureWritable(size_t a_len)
    {
        if (WritableBytes() < a_len)
            MakeSpace(a_len);
    }
    void HasWritten(size_t a_len)
    {
        m_write_idx_ += a_len;
    }

    /// @brief 从 fd 读取数据（readv 优化：栈上临时缓冲 + 内部可写区）
    /// @return >0 读取字节数；0 对端关闭；<0 出错（errno 存入 a_saved_errno）
    ssize_t ReadFd(int a_fd, int* a_saved_errno)
    {
        char extrabuf[65536];
        struct iovec vec[2];
        const size_t writable = WritableBytes();
        vec[0].iov_base = BeginWrite();
        vec[0].iov_len = writable;
        vec[1].iov_base = extrabuf;
        vec[1].iov_len = sizeof(extrabuf);

        const ssize_t n = ::readv(a_fd, vec, 2);
        if (n < 0)
        {
            *a_saved_errno = errno;
        }
        else if (static_cast<size_t>(n) <= writable)
        {
            HasWritten(n);
        }
        else
        {
            m_write_idx_ = m_buffer_.size();
            Append(extrabuf, n - writable);
        }
        return n;
    }

    void Clear()
    {
        RetrieveAll();
    }

private:
    char* begin() { return &*m_buffer_.begin(); }
    const char* begin() const { return &*m_buffer_.begin(); }

    void MakeSpace(size_t a_len)
    {
        if (WritableBytes() + PrependableBytes() < a_len + kCheapPrepend)
        {
            ////空间不足，直接扩容
            m_buffer_.resize(m_write_idx_ + a_len);
        }
        else
        {
            ////把可读数据前移到 kCheapPrepend 处，腾出尾部空间
            const size_t readable = ReadableBytes();
            std::copy(begin() + m_read_idx_, begin() + m_write_idx_, begin() + kCheapPrepend);
            m_read_idx_ = kCheapPrepend;
            m_write_idx_ = m_read_idx_ + readable;
        }
    }
};

}//////namespace su
#endif
