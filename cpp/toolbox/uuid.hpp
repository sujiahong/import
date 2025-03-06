////////
////uuid生成器
////////

#ifndef _UUID_HPP_
#define _UUID_HPP_

#include "time_function.hpp"
#include <atomic>
#include <chrono>
#include <stdexcept>

namespace su
{

//////单进程下生成uuid，保证id不重复, 时间回拨可能会重复, 速度快
/* atomic 操作
store 赋值
load  读取
exchange  原子变量赋值
a.compare_exchange_weak(b, c) a当前值，b为期望值，c为新值
    a == b时：返回真，并把c赋值给a
    a != b时：返回假，并把a赋值给b
compare_exchange_strong
*/
static uint64_t sp_uuid()
{
    uint64_t time = NanoTime();
    static std::atomic<uint64_t> atm_last_time{time};
    uint64_t prev = atm_last_time.load(std::memory_order_relaxed);
    do {
        if (time > prev)
        {
            if (atm_last_time.compare_exchange_weak(prev, time))
                return time;
        }
        else
        {
            if (atm_last_time.compare_exchange_weak(prev, prev + 1))
                return prev + 1;
        }
    }while(1);
}


class SnowflakeID {
private:
    static constexpr uint64_t EPOCH = 1577836800000ULL; // 2020-01-01
    static constexpr int TIMESTAMP_BITS = 41;
    static constexpr int MACHINE_BITS = 10;
    static constexpr int SEQUENCE_BITS = 12;

    std::atomic<uint64_t> last_timestamp_{0};
    uint16_t machine_id_;
    uint16_t sequence_{0};

public:
    explicit SnowflakeID(uint16_t machine_id) 
        : machine_id_(machine_id & ((1 << MACHINE_BITS) - 1)) {}

    uint64_t Generate() {
        uint64_t curr_ts = GetTimestamp();
        uint64_t last_ts = last_timestamp_.load(std::memory_order_relaxed);

        // 处理时间回拨（超过1秒视为严重错误）
        if (curr_ts < (last_ts >> (MACHINE_BITS + SEQUENCE_BITS))) {
            if ((last_ts >> (MACHINE_BITS + SEQUENCE_BITS)) - curr_ts > 1000) {
                throw std::runtime_error("Clock moved backwards over 1 second");
            }
            curr_ts = last_ts >> (MACHINE_BITS + SEQUENCE_BITS);
        }

        if (curr_ts == (last_ts >> (MACHINE_BITS + SEQUENCE_BITS))) {
            sequence_ = (sequence_ + 1) & ((1 << SEQUENCE_BITS) - 1);
            if (sequence_ == 0) {
                curr_ts = WaitNextMillis(curr_ts);
            }
        } else {
            sequence_ = 0;
        }

        uint64_t id = (curr_ts << (MACHINE_BITS + SEQUENCE_BITS)) 
                    | (machine_id_ << SEQUENCE_BITS) 
                    | sequence_;

        last_timestamp_.store(
            (curr_ts << (MACHINE_BITS + SEQUENCE_BITS)) | 
            (machine_id_ << SEQUENCE_BITS) | 
            sequence_,
            std::memory_order_relaxed
        );

        return id;
    }

private:
    uint64_t GetTimestamp() const {
        return std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()
        ).count() - EPOCH;
    }

    uint64_t WaitNextMillis(uint64_t last_ts) const {
        uint64_t ts = GetTimestamp();
        while (ts <= last_ts) {
            std::this_thread::yield();
            ts = GetTimestamp();
        }
        return ts;
    }
};
}


#endif