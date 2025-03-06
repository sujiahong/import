#ifndef __BASE_MACRO_H__
#define __BASE_MACRO_H__

#ifndef offsetof        
#define offsetof(TYPE, MEMBER) ((size_t) ((char*)&((TYPE *)0)->MEMBER))      /////计算结构体内 member 的偏移量
#endif

#ifndef containerof_new
#define containerof_new(ptr, type, memb) ((type*) ((char*)((const typeof(((type*)0) -> memb)*)(ptr)) - offsetof(type,memb))) /////返回结构体类型的指针
#endif

/////第一个线程可以过去，后面的线程等待
#define spin_lock(lk) while(!__sync_bool_compare_and_swap(lk, 0, 1)) { \
                          do { \
                                // 自旋等待，直到获取锁
                                #if defined(__x86_64__) || defined(__i386__) \
                                    __builtin_ia32_pause();  \ // x86架构优化：降低CPU功耗
                                #elif defined(__aarch64__) \
                                    __asm__ __volatile__("yield" ::: "memory"); \ // ARM架构优化
                                #endif \
                          } while(*lk); \
                    }
#define spin_unlock(lk) __sync_lock_release(lk);

////内存读，写屏障
#define read_mem_barrier do {                         \
    #if defined(__x86_64__) || defined(__i386__)        \
        __asm__ __volatile__("lfence":::"memory");      \
    #elif  defined(__aarch64__)                         \
        __asm__ __volatile__("dmb ishld":::"memory");   \
    #else                                               \
        __atomic_thread_fence(__ATOMIC_ACQUIRE);        \
    #endif                                              \
} while(0);
#define write_mem_barrier do {                        \
    #if defined(__x86_64__) || defined(__i386__)        \
        __asm__ __volatile__("sfence":::"memory");      \
    #elif  defined(__aarch64__)                         \
        __asm__ __volatile__("dmb ishst":::"memory");   \
    #else                                               \
        __atomic_thread_fence(__ATOMIC_RELEASE);        \
    #endif                                              \
} while(0);
// 全内存屏障（Full Barrier）
#define full_mem_barrier do { \
    #if defined(__x86_64__) || defined(__i386__)       \
        __asm__ __volatile__("mfence" ::: "memory");    \
    #elif defined(__aarch64__)                         \
        __asm__ __volatile__("dmb ish" ::: "memory");   \
    #else                                               \
        __atomic_thread_fence(__ATOMIC_SEQ_CST);        \
    #endif                                              \
} while(0)

/*
// 原型
long __builtin_expect(long exp, long value);
提示编译器 exp == value 的概率很高
通过调整指令顺序减少分支预测惩罚（Branch Misprediction Penalty）
优化效果：编译器会将代码块A紧接条件判断编译，减少跳转指令
实测数据：在 Linux 内核等高频判断场景可提升 10-20% 性能

适用性优先级
只在满足以下条件时使用：
分支条件在 性能关键路径 上
分支概率存在 明显偏向性（如 95% vs 5%）

验证方法
通过 gcc -S -O2 生成汇编，确认代码布局变化
使用 perf stat -e branch-misses 检测分支预测失败率
*/
#ifndef likely
#if defined(__GNUC__) || defined(__clang__)
    #define likely(cond)   (__builtin_expect(!!(cond), 1))  // 暗示条件大概率成立
    #define unlikely(cond) (__builtin_expect(!!(cond), 0))  // 暗示条件大概率不成立
#else  // 其他编译器兼容
    #define likely(cond)   (cond)
    #define unlikely(cond) (cond)
#endif
#endif

#endif