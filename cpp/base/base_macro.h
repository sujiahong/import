#ifndef __BASE_MACRO_H__
#define __BASE_MACRO_H__

#ifndef offsetof        
#define offsetof(TYPE, MEMBER) ((size_t) ((char*)&((TYPE *)0)->MEMBER))      /////计算结构体内 member 的偏移量
#endif

#ifndef containerof_new
#define containerof_new(ptr, type, memb) ((type*) ((char*)((const typeof(((type*)0) -> memb)*)(ptr)) - offsetof(type,memb))) /////返回结构体类型的指针
#endif



#endif