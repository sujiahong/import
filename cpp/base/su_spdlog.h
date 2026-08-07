#ifndef SU_SPDLOG_H
#define SU_SPDLOG_H

#include <memory>
#include <mutex>
#include <string>

#include "../spdlog/async.h"
#include "../spdlog/common.h"
#include "../spdlog/spdlog.h"
#include "../spdlog/sinks/basic_file_sink.h"
#include "../spdlog/sinks/rotating_file_sink.h"

namespace su
{

class SuSpdlog
{
private:
    static constexpr const char* kAsyncRotateLoggerName = "async_rotate_logger";
    static constexpr const char* kRotateLoggerName = "spd_rotate_logger";
    static constexpr const char* kBasicLoggerName = "spd_basic_logger";
    static constexpr const char* kLogPattern ="%# %@[%Y-%m-%d %H:%M:%S.%e][%^%l%$][%t] %!, %v";
    enum
    {
        kAsyncQueueSize = 8192,
        kAsyncThreadCount = 4,
        kMaxFileSize = 1024 * 1024 * 300,
        kMaxFiles = 100
    };
private:
    std::shared_ptr<spdlog::logger> async_rotate_logger_ = nullptr;
    std::shared_ptr<spdlog::logger> spd_rotate_logger_ = nullptr;
    std::shared_ptr<spdlog::logger> spd_basic_logger_ = nullptr;
private:
    SuSpdlog() = default;
    ~SuSpdlog() = default;
public:
    void InitSuSpdlog(const std::string& log_path = "./log", const std::string& log_level = "trace")
    {
        std::lock_guard<std::mutex> guard(GetMutex());
        spdlog::level::level_enum spdlog_level = ToLevel(log_level);

        if (!spdlog::get(kAsyncRotateLoggerName))
        {
            spdlog::init_thread_pool(kAsyncQueueSize, kAsyncThreadCount);
        }

        async_rotate_logger_ = GetOrCreateAsyncRotateLogger(log_path);
        spd_rotate_logger_ = GetOrCreateRotateLogger(log_path);
        spd_basic_logger_ = GetOrCreateBasicLogger(log_path);

        SetupLogger(async_rotate_logger_, spdlog_level);
        SetupLogger(spd_rotate_logger_, spdlog_level);
        SetupLogger(spd_basic_logger_, spdlog_level);

        spdlog::set_default_logger(async_rotate_logger_);
        spdlog::flush_on(spdlog::level::err);
    }

    void SetSpdlogLevel(const std::string& level)
    {
        std::lock_guard<std::mutex> guard(GetMutex());
        spdlog::level::level_enum spdlog_level = ToLevel(level);

        SetLoggerLevel(kAsyncRotateLoggerName, spdlog_level);
        SetLoggerLevel(kRotateLoggerName, spdlog_level);
        SetLoggerLevel(kBasicLoggerName, spdlog_level);
        spdlog::set_level(spdlog_level);
    }

    void SetSpdlogLevelByLoggerName(const std::string& logger_name, spdlog::level::level_enum level)
    {
        std::shared_ptr<spdlog::logger> logger = spdlog::get(logger_name);
        if (logger)
        {
            logger->set_level(level);
        }
    }
    inline std::shared_ptr<spdlog::logger> get_logger()
    {
        if (async_rotate_logger_)
        {
            return async_rotate_logger_;
        }
        return spdlog::default_logger();
    }

    void async_trace(const std::string& msg)
    {
        LogString(async_rotate_logger_, spdlog::level::trace, msg);
    }

    void async_debug(const std::string& msg)
    {
        LogString(async_rotate_logger_, spdlog::level::debug, msg);
    }

    void async_info(const std::string& msg)
    {
        LogString(async_rotate_logger_, spdlog::level::info, msg);
    }

    void async_warn(const std::string& msg)
    {
        LogString(async_rotate_logger_, spdlog::level::warn, msg);
    }

    void async_error(const std::string& msg)
    {
        LogString(async_rotate_logger_, spdlog::level::err, msg);
    }

private:
    static spdlog::level::level_enum ToLevel(const std::string& level)
    {
        spdlog::level::level_enum spdlog_level = spdlog::level::from_str(level);
        if (spdlog_level == spdlog::level::off && level != "off")
        {
            return spdlog::level::trace;
        }
        return spdlog_level;
    }

    static std::string JoinPath(const std::string& path, const std::string& filename)
    {
        if (path.empty())
        {
            return filename;
        }
        if (path[path.size() - 1] == '/')
        {
            return path + filename;
        }
        return path + "/" + filename;
    }

    static void SetupLogger(const std::shared_ptr<spdlog::logger>& logger,
                            spdlog::level::level_enum level)
    {
        if (!logger)
        {
            return;
        }
        logger->set_pattern(kLogPattern);
        logger->set_level(level);
    }

    static std::shared_ptr<spdlog::logger> GetOrCreateAsyncRotateLogger(const std::string& log_path)
    {
        std::shared_ptr<spdlog::logger> logger = spdlog::get(kAsyncRotateLoggerName);
        if (!logger)
        {
            logger = spdlog::create_async<spdlog::sinks::rotating_file_sink_mt>(
                kAsyncRotateLoggerName,
                JoinPath(log_path, "async_rotate.log"),
                kMaxFileSize,
                kMaxFiles);
        }
        return logger;
    }

    static std::shared_ptr<spdlog::logger> GetOrCreateRotateLogger(const std::string& log_path)
    {
        std::shared_ptr<spdlog::logger> logger = spdlog::get(kRotateLoggerName);
        if (!logger)
        {
            logger = spdlog::rotating_logger_mt(
                kRotateLoggerName,
                JoinPath(log_path, "rotate.log"),
                kMaxFileSize,
                kMaxFiles);
        }
        return logger;
    }

    static std::shared_ptr<spdlog::logger> GetOrCreateBasicLogger(const std::string& log_path)
    {
        std::shared_ptr<spdlog::logger> logger = spdlog::get(kBasicLoggerName);
        if (!logger)
        {
            logger = spdlog::basic_logger_mt(
                kBasicLoggerName,
                JoinPath(log_path, "basic.log"));
        }
        return logger;
    }

    inline void LogString(std::shared_ptr<spdlog::logger> logger, spdlog::level::level_enum level, const std::string& msg)
    {
        if (logger)
        {
            logger->log(level, "{}", msg);
        }
    }

    static std::mutex& GetMutex()
    {
        static std::mutex mutex;
        return mutex;
    }
};

inline SuSpdlog& GetSuSpdlog()
{
    static SuSpdlog su_spdlog;
    return su_spdlog;
}

} // namespace su

#define SU_LOG_TRACE(...) SPDLOG_LOGGER_TRACE(::su::GetSuSpdlog().get_logger(), __VA_ARGS__)
#define SU_LOG_DEBUG(...) SPDLOG_LOGGER_DEBUG(::su::GetSuSpdlog().get_logger(), __VA_ARGS__)
#define SU_LOG_INFO(...) SPDLOG_LOGGER_INFO(::su::GetSuSpdlog().get_logger(), __VA_ARGS__)
#define SU_LOG_WARN(...) SPDLOG_LOGGER_WARN(::su::GetSuSpdlog().get_logger(), __VA_ARGS__)
#define SU_LOG_ERROR(...) SPDLOG_LOGGER_ERROR(::su::GetSuSpdlog().get_logger(), __VA_ARGS__)
#define SU_LOG_CRITICAL(...) SPDLOG_LOGGER_CRITICAL(::su::GetSuSpdlog().get_logger(), __VA_ARGS__)

#endif // SU_SPDLOG_H
