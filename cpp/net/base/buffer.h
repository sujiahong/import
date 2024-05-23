#ifndef __BUFFER_H__
#define __BUFFER_H__

#include "../../toolbox/original_dependence.hpp"

#include <vector>

namespace su
{
class Buffer: public Copyable {
private:
    std::vector<char> m_buffer_;
    

};

}
#endif