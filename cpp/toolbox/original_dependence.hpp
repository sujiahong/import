#ifndef _ORIGINAL_DEPENDENCE_HPP_
#define _ORIGINAL_DEPENDENCE_HPP_

namespace su
{
////继承，实现类不可拷贝，赋值
class Noncopyable
{
protected:
    Noncopyable();
    ~Noncopyable();
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



}



#endif


n