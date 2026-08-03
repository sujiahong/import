// Callback helpers for the network layer.

#ifndef SU_NET_CALLBACKS_H
#define SU_NET_CALLBACKS_H

#include <cassert>
#include <cstddef>
#include <functional>
#include <memory>

#include "base_define.h"

namespace su
{
template<typename To, typename From>
inline std::shared_ptr<To> down_pointer_cast(const std::shared_ptr<From>& f)
{
#ifndef NDEBUG
    assert(!f || dynamic_cast<To*>(f.get()) != nullptr);
#endif
    return std::static_pointer_cast<To>(f);
}

using TimerCallback = EVENT_CALLBACK_TYPE;
using ConnectionCallback = CONNECTION_CALLBACK_TYPE;
using CloseCallback = CLOSE_CALLBACK_TYPE;
using WriteCompleteCallback = WRITE_COMPLETE_CALLBACK_TYPE;
using MessageCallback = MESSAGE_CALLBACK_TYPE;
using TcpConnectionPtr = TCP_CONNECTION_PTR;
using HighWaterMarkCallback =
    std::function<void(const TcpConnectionPtr&, std::size_t)>;
} // namespace su

#endif // SU_NET_CALLBACKS_H
