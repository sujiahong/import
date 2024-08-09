#ifndef __BUFFER_H__
#define __BUFFER_H__

#include "../../toolbox/original_dependence.hpp"

#include <vector>

namespace su
{
class Buffer: public Copyable {
public:
    static const size_t kCheapPrepend = 8;
    static const size_t kInitialSize = 4096;
private:
    std::vector<char> m_buffer_;
    size_t m_read_idx_;
    size_t m_write_idx_;
public:
    Buffer(size_t a_size = 1024);
    ~Buffer();

    void Clear()
    {
        m_read_idx_ = 0;
        m_write_idx_ = 0;
    }
    
};

}
#endif