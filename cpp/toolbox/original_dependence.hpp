#ifndef _ORIGINAL_DEPENDENCE_HPP_
#define _ORIGINAL_DEPENDENCE_HPP_

#include <pthread.h>

namespace su
{
////继承，实现类不可拷贝，赋值
class Noncopyable
{
protected:
    Noncopyable(){};
    ~Noncopyable(){};
private:
    Noncopyable(const Noncopyable&);
    const Noncopyable& operator=(const Noncopyable&);
};

/////继承
template<typename T>
class OnlyOneInstance
{
public:
    static T& Instance()
    {
        static T intance;////////在 C++ 11 之后，被 static修饰的函数内部变量可以保证是线程安全的
        return intance; 
    }
private:
    OnlyOneInstance(OnlyOneInstance<T>&&){}
    OnlyOneInstance(const OnlyOneInstance<T>&){}
    const OnlyOneInstance<T>& operator= (const OnlyOneInstance<T>&){}
protected:
    OnlyOneInstance(){}
    virtual ~OnlyOneInstance(){}
};
/////继承
/////单例类
template<typename T>
class Singleton :public Noncopyable
{
public:
    static T& Instance()
    {
        phread_once(&m_ponce_, &Singleton::Init);
        return *m_value_;
    }

    static void Init()
    {
        m_value_ = new T();
    }
private:
    static pthread_once_t m_ponce_;
    static T* m_value_;
};

template<typename T>
pthread_once_t Singleton<T>::m_ponce_ = PTHREAD_ONCE_INIT;

template<typename T>
T* Singleton<T>::m_value_ = 0;

}



#endif