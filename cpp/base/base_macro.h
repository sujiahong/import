#ifndef __BASE_MACRO_H__
#define __BASE_MACRO_H__

#include <cstddef>

#ifndef offsetof        
#define offsetof(TYPE, MEMBER) ((size_t) ((char*)&((TYPE *)0)->MEMBER))      /////计算结构体内 member 的偏移量
#endif

#ifndef containerof_new
#define containerof_new(ptr, type, memb) ((type*) ((char*)((const typeof(((type*)0) -> memb)*)(ptr)) - offsetof(type,memb))) /////返回结构体类型的指针
#endif

namespace su
{
inline void CpuRelax() noexcept
{
#if defined(__x86_64__) || defined(__i386__)
    __builtin_ia32_pause();
#elif defined(__aarch64__) || defined(__arm__)
    __asm__ __volatile__("yield" ::: "memory");
#elif defined(__riscv)
    __asm__ __volatile__(".insn i 0x0f, 0, x0, x0, 0x010" ::: "memory");
#else
    __atomic_signal_fence(__ATOMIC_ACQUIRE);
#endif
}

inline void ReadMemBarrier() noexcept
{
    __atomic_thread_fence(__ATOMIC_ACQUIRE);
}

inline void WriteMemBarrier() noexcept
{
    __atomic_thread_fence(__ATOMIC_RELEASE);
}

inline void FullMemBarrier() noexcept
{
    __atomic_thread_fence(__ATOMIC_SEQ_CST);
}
} // namespace su

/////第一个线程可以过去，后面的线程等待
#define spin_lock(lk) do {                                             \
    auto* const spin_lock_ptr = (lk);                                 \
    int spin_lock_expected = 0;                                       \
    while (!__atomic_compare_exchange_n(                              \
        spin_lock_ptr, &spin_lock_expected, 1, false,                 \
        __ATOMIC_ACQUIRE, __ATOMIC_RELAXED)) {                        \
        spin_lock_expected = 0;                                       \
        while (__atomic_load_n(spin_lock_ptr, __ATOMIC_RELAXED)) {     \
            ::su::CpuRelax();                                         \
        }                                                              \
    }                                                                  \
} while (0)

#define spin_unlock(lk) do {                                           \
    __atomic_store_n((lk), 0, __ATOMIC_RELEASE);                       \
} while (0)

////内存读，写屏障
#define read_mem_barrier do { ::su::ReadMemBarrier(); } while (0)
#define write_mem_barrier do { ::su::WriteMemBarrier(); } while (0)
// 全内存屏障（Full Barrier）
#define full_mem_barrier do { ::su::FullMemBarrier(); } while (0)

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


/*
使用 C 和 C++ 的用户来说，可能有三种不同的编译选项：
其中 CFLAGS 对应 C 语言特有的编译选项、
CXXFLAGS 对应是 C++ 特有的编译选项、
CPPFLAGS 则对应 C 和 C++ 共有的编译选项。
但是在链接阶段，C 和 C++ 的链接选项是通用的，因此这个时候已经不再有 C 和 C++ 语言的区别，
它们的目标文件的类型是相同的。
*/

#endif
