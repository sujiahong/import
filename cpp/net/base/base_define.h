#ifndef BASE_DEFINE_H
#define BASE_DEFINE_H

#include <functional>
#include <memory>
#include <string>


namespace su
{
class Connection;

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

typedef std::function<void(int, std::string, unsigned short)> NEW_CONNECTION_CALLBACK_TYPE;

typedef std::function<void()> EVENT_CALLBACK_TYPE;
typedef std::function<void(unsigned int)> READ_EVENT_CALLBACK_TYPE;

typedef std::function<void(Connection*)>  CONNECTION_CALLBACK_TYPE;

} ///namespace su

#endif