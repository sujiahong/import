/////////////////////
////os socket 封装
////////////////////

#ifndef __SOCKET_H__
#define __SOCKET_H__

#include "../../base/original_base.h"
#include "../../base/su_spdlog.h"
#include <unistd.h>
#include <arpa/inet.h>
#include <cstdio>
#include <cstring>
#include <sys/socket.h>
#include <sys/types.h>
#include <fcntl.h>
#include <errno.h>
#include <netinet/tcp.h>
#include <sys/uio.h>  // readv

#include <string>
namespace su
{
class Socket: public Noncopyable
{
private:
    int m_sock_ver_;//////标记ipv4 AF_INET or ipv6 AF_INET6
    int m_type_; //SOCK_STREAM   SOCK_DGRAM
    int m_sock_fd_;
public:
    Socket() = delete;
    Socket(int a_sa_family, int a_type):
        m_sock_ver_(a_sa_family), m_type_(a_type), m_sock_fd_(-1)
    {
        m_sock_fd_ = Create(a_sa_family, a_type);
    }
    Socket(int a_fd):m_sock_ver_(0),m_type_(0),m_sock_fd_(a_fd)
    {}
    ~Socket()
    {
        ::close(m_sock_fd_);
    }
public:
    inline int Fd() const {return m_sock_fd_;}
private:
    static void PrintSystemError(const char* a_op, int a_err_no)
    {
        SU_SYNC_LOG_ERROR("{} failed: errno={} ({})", a_op, a_err_no, strerror(a_err_no));
    }
public:
    int Create(int a_sa_family, int a_type)
    {
        m_sock_ver_ = a_sa_family;
        m_type_ = a_type;
        int sock_fd = ::socket(a_sa_family, m_type_, 0);
        if (sock_fd < 0)
        {
            m_sock_fd_ = -1;
            int en = errno;
            PrintSystemError("socket", en);
            return -1;
        }
        if (m_sock_fd_ >= 0)
            ::close(m_sock_fd_);
        m_sock_fd_ = sock_fd;
        return sock_fd;
    }
    void Bind(const std::string a_ip, const unsigned short a_port)
    {
        struct sockaddr_storage local_addr{};
        socklen_t len = 0;
        if (m_sock_ver_ == AF_INET)
        {
            auto* addr = reinterpret_cast<sockaddr_in*>(&local_addr);
            addr->sin_family = AF_INET;
            addr->sin_addr.s_addr = a_ip.empty() ? htonl(INADDR_ANY) : inet_addr(a_ip.c_str());
            addr->sin_port = htons(a_port);
            len = sizeof(sockaddr_in);
        }
        else if (m_sock_ver_ == AF_INET6) 
        {
            auto* addr6 = reinterpret_cast<sockaddr_in6*>(&local_addr);
            addr6->sin6_family = AF_INET6;
            addr6->sin6_port = htons(a_port);
            // addr6->sin6_addr = in6addr_any; // 或 inet_pton(AF_INET6, ...)
            int inet_ret = inet_pton(AF_INET6, a_ip.c_str(), &addr6->sin6_addr);
            if (inet_ret == 0)
            {
                SU_SYNC_LOG_WARN("inet_pton(AF_INET6) failed: invalid ip={}", a_ip);
            }
            else if (inet_ret < 0)
            {
                PrintSystemError("inet_pton(AF_INET6)", errno);
            }
            len = sizeof(sockaddr_in6);
        }
        int ret = ::bind(m_sock_fd_, (struct sockaddr*)&local_addr, len);
        if (ret < 0)
        {
            PrintSystemError("bind", errno);
        }
    }
    void Bind(const unsigned short a_port)
    {
        struct sockaddr_storage local_addr{};
        socklen_t len = 0;
        if (m_sock_ver_ == AF_INET)
        {
            auto* addr = reinterpret_cast<sockaddr_in*>(&local_addr);
            addr->sin_family = AF_INET;
            addr->sin_addr.s_addr = htonl(INADDR_ANY);
            addr->sin_port = htons(a_port);
            len = sizeof(sockaddr_in);
        }
        else if (m_sock_ver_ == AF_INET6) 
        {
            auto* addr6 = reinterpret_cast<sockaddr_in6*>(&local_addr);
            addr6->sin6_family = AF_INET6;
            addr6->sin6_port = htons(a_port);
            addr6->sin6_addr = in6addr_any; // 或 inet_pton(AF_INET6, ...)
            len = sizeof(sockaddr_in6);
        }
        int ret = ::bind(m_sock_fd_, (struct sockaddr*)&local_addr, len);
        if (ret < 0)
        {
            PrintSystemError("bind", errno);
        }
    }
    int Listen()
    {
        int ret = ::listen(m_sock_fd_, 128);
        if (ret < 0)
        {
            PrintSystemError("listen", errno);
            return -1;
        }
        return 0;
    }
    int Accept(std::string& a_peer_ip, unsigned short& a_peer_port)////返回对端的ip，port
    {
        struct sockaddr peer_addr;
        socklen_t addr_len = sizeof(peer_addr);
        int conn_fd = ::accept(m_sock_fd_, &peer_addr, &addr_len);//, SOCK_NONBLOCK|SOCK_CLOEXEC);
        if (conn_fd < 0)
        {
            int err_no = errno;
            PrintSystemError("accept", err_no);
            switch (err_no)
            {
            case EAGAIN:
            case ECONNABORTED:
            case EINTR:
            case EPROTO:
            case EPERM:
            case EMFILE:
                /////per-process lmit of open file desc
                break;
            case EBADF:
            case EFAULT:
            case EINVAL:
            case ENOBUFS:
            case ENOMEM:
            case ENOTSOCK:
            case EOPNOTSUPP:
                /////unexpected error ::accept <<err_no
                break;
            default:
                ////unknown error of accept
                break;
            }
        }
        else
            ExtractIPAndPortFromSockAddr(&peer_addr, a_peer_ip, a_peer_port);
        if (conn_fd >= 0)
            Socket::SetNoBlockAndCloseOnExec(conn_fd); /////set nonblocking and close on exec
        return conn_fd;
    }
    int Connect(std::string a_ip, unsigned short a_port)
    {
        struct sockaddr_in peer_addr;
        peer_addr.sin_family = m_sock_ver_;
        peer_addr.sin_addr.s_addr = inet_addr(a_ip.c_str());
        peer_addr.sin_port = htons(a_port);
        int ret = ::connect(m_sock_fd_, (struct sockaddr*)&peer_addr, sizeof(peer_addr));
        if (ret < 0)
        {
            PrintSystemError("connect", errno);
        }
        return ret;
    }
    void ShutdownWrite()
    {
        if (::shutdown(m_sock_fd_, SHUT_WR)<0)
        {
            PrintSystemError("shutdown", errno);
        }
    }
    void SetTcpNoDelay(bool a_on)
    {
        int opval = a_on ? 1 : 0;
        if (::setsockopt(m_sock_fd_, IPPROTO_TCP, TCP_NODELAY, &opval, sizeof(opval)) < 0)
        {
            PrintSystemError("setsockopt(TCP_NODELAY)", errno);
        }
    }
    void SetReuseAddr(bool a_on)
    {
        int opval = a_on ? 1 : 0;
        if (::setsockopt(m_sock_fd_, SOL_SOCKET, SO_REUSEADDR, &opval, sizeof(opval)) < 0)
        {
            PrintSystemError("setsockopt(SO_REUSEADDR)", errno);
        }
    }
    void SetReusePort(bool a_on)
    {
        int opval = a_on ? 1 : 0;
        if (::setsockopt(m_sock_fd_, SOL_SOCKET, SO_REUSEPORT, &opval, sizeof(opval)) < 0)
        {
            PrintSystemError("setsockopt(SO_REUSEPORT)", errno);
        }
    }
    void SetKeepAlive(bool a_on)
    {
        int opval = a_on ? 1 : 0;
        if (::setsockopt(m_sock_fd_, SOL_SOCKET, SO_KEEPALIVE, &opval, sizeof(opval)) < 0)
        {
            PrintSystemError("setsockopt(SO_KEEPALIVE)", errno);
        }
    }
    static void SetNoBlockAndCloseOnExec(int a_sock_fd)
    {
        int flags = ::fcntl(a_sock_fd, F_GETFL, 0);
        if (flags < 0)
        {
            PrintSystemError("fcntl(F_GETFL)", errno);
            return;
        }
        flags |= O_NONBLOCK;
        if (::fcntl(a_sock_fd, F_SETFL, flags) < 0)
        {
            PrintSystemError("fcntl(F_SETFL)", errno);
        }

        flags = ::fcntl(a_sock_fd, F_GETFD, 0);
        if (flags < 0)
        {
            PrintSystemError("fcntl(F_GETFD)", errno);
            return;
        }
        flags |= FD_CLOEXEC;
        if (::fcntl(a_sock_fd, F_SETFD, flags) < 0)
        {
            PrintSystemError("fcntl(F_SETFD)", errno);
        }
    }
    static void ExtractIPAndPortFromSockAddr(struct sockaddr* a_addr, std::string& a_ip, unsigned short& a_port)
    {
        struct sockaddr_in* addr_in = reinterpret_cast<struct sockaddr_in*>(a_addr);
        char ip_str[INET_ADDRSTRLEN];
        inet_ntop(AF_INET, &(addr_in->sin_addr.s_addr), ip_str, sizeof(ip_str));
        a_port = ntohs(addr_in->sin_port);
        a_ip = ip_str;
    }
    static int GetSelfIPAndPort(int a_fd, std::string& a_ip, unsigned short& a_port)
    {
        struct sockaddr addr;
        socklen_t addrLen = sizeof(addr);
        int ret = ::getsockname(a_fd, &addr, &addrLen);
        if (ret != 0){
            PrintSystemError("getsockname", errno);
            return ret;
        }
        ExtractIPAndPortFromSockAddr(&addr, a_ip, a_port);
        return 0;
    }
    static int GetPeerIPAndPort(int a_fd, std::string& a_ip, unsigned short& a_port)
    {
        struct sockaddr addr;
        socklen_t addrLen = sizeof(addr);
        int ret = ::getpeername(a_fd, &addr, &addrLen);
        if (ret != 0){
            PrintSystemError("getpeername", errno);
            return ret;
        }
        ExtractIPAndPortFromSockAddr(&addr, a_ip, a_port);
        return 0;
    }
    static long int Read(int a_fd, void* a_buf, size_t a_len)
    {   
        long int ret = (long int)::read(a_fd, a_buf, a_len);
        if (ret < 0)
        {
            PrintSystemError("read", errno);
        }
        return ret;
    }
    static long int Readv(int a_fd, const struct iovec* a_iov, int a_iovcnt)
    {
        long int ret = (long int)::readv(a_fd, a_iov, a_iovcnt);
        if (ret < 0)
        {
            PrintSystemError("readv", errno);
        }
        return ret;
    }
    static long int Write(int a_fd, const void* a_buf, size_t a_len)
    {
        long int ret = (long int)::write(a_fd, a_buf, a_len);
        if (ret < 0)
        {
            PrintSystemError("write", errno);
        }
        return ret;
    }
};
}

#endif
