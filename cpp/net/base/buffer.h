#ifndef __BUFFER_H__
#define __BUFFER_H__

#include "../../toolbox/original_dependence.hpp"

#include <vector>

namespace su
{
class Buffer: public Copyable {
private:
    std::vector<char> m_buffer_;
    size_t m_read_idx_;
    size_t m_write_idx_;
public:


    
};

}
#endif