#ifndef BASE_DEFINE_H
#define BASE_DEFINE_H

#include <functional>
#include <memory>
#include <string>
#include <unordered_map>

namespace su
{
class Connection;
class Buffer;
template<typename T>
inline T* get_pointer(const std::shared_ptr<T>& ptr)
{
  return ptr.get();
}

template<typename T>
inline T* get_pointer(const std::unique_ptr<T>& ptr)
{
  return ptr.get();
}

typedef std::function<void(int, const std::string&, unsigned short)> NEW_CONNECTION_CALLBACK_TYPE;

typedef std::function<void()> EVENT_CALLBACK_TYPE;
typedef std::function<void(unsigned int)> READ_EVENT_CALLBACK_TYPE;


typedef std::shared_ptr<Connection> TCP_CONNECTION_PTR;
typedef std::unordered_map<int, TCP_CONNECTION_PTR> CONNECTION_MAP_TYPE;

typedef std::function<void (const TCP_CONNECTION_PTR&)> CONNECTION_CALLBACK_TYPE;
typedef std::function<void (const TCP_CONNECTION_PTR&)> CLOSE_CALLBACK_TYPE;
typedef std::function<void (const TCP_CONNECTION_PTR&)> WRITE_COMPLETE_CALLBACK_TYPE;
typedef std::function<void (const TCP_CONNECTION_PTR&, Buffer*, unsigned int)> MESSAGE_CALLBACK_TYPE;


} ///namespace su

#endif