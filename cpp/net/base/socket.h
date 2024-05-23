/////////////////////
////os socket 封装
////////////////////

#ifndef __SOCKET_H__
#define __SOCKET_H__

#include "../toolbox/original_dependence.hpp"
#include <unistd.h>
#include <arpa/inet.h>
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
    Socket(int a_sa_family, int a_type):m_sock_ver_(a_sa_family),m_type_(a_type)
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
    int Create(int a_sa_family, int a_type)
    {
        m_sock_ver_ = a_sa_family;
        m_type_ = a_type;
        int sock_fd = ::socket(a_sa_family, m_type_, 0);
        if (m_sock_fd_ < 0)
            m_sock_fd_ = sock_fd;
        else
        {
            ::close(m_sock_fd_);
            m_sock_fd_ = sock_fd;
        }
        return sock_fd;
    }
    void Bind(const std::string a_ip, const unsigned short a_port)
    {
        struct sockaddr_in local_addr;
        local_addr.sin_family = AF_INET;
        local_addr.sin_addr.s_addr = inet_addr(a_ip.c_str());
        local_addr.sin_port = htons(a_port);
        int ret = ::bind(m_sock_fd_, (struct sockaddr*)&local_addr, sizeof(local_addr));
        if (ret < 0)
        {
            ///打印日志
        }
    }
    void Bind(const unsigned short a_port)
    {
        struct sockaddr_in local_addr;
        local_addr.sin_family = AF_INET;
        local_addr.sin_addr.s_addr = htonl(INADDR_ANY); 
        local_addr.sin_port = htons(a_port);
        int ret = ::bind(m_sock_fd_, (struct sockaddr*)&local_addr, sizeof(local_addr));
        if (ret < 0)
        {
            ///打印日志
        }
    }
    void Listen()
    {
        int ret = ::listen(m_sock_fd_, 128);
        if (ret < 0)
        {
            ///打印日志
        }
    }
    int Accept(std::string& a_peer_ip, unsigned short& a_peer_port)////返回对端的ip，port
    {
        struct sockaddr peer_addr;
        socklen_t addr_len = sizeof(peer_addr);
        int conn_fd = ::accept(m_sock_fd_, &peer_addr, &addr_len);//, SOCK_NONBLOCK|SOCK_CLOEXEC);
        if (conn_fd < 0)
        {
            int err_no = errno;
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
        Socket::SetNoBlockAndCloseOnExec(conn_fd); /////set nonblocking and close on exec
        return conn_fd;
    }
    int Connect(std::string a_ip, unsigned short a_port)
    {
        struct sockaddr_in peer_addr;
        peer_addr.sin_family = m_sock_ver_;
        peer_addr.sin_addr.s_addr = inet_addr(a_ip.c_str());
        peer_addr.sin_port = htons(a_port);
        return ::connect(m_sock_fd_, (struct sockaddr*)&peer_addr, sizeof(peer_addr));
    }
    void ShutdownWrite()
    {
        if (::shutdown(m_sock_fd_, SHUT_WR)<0)
        {
            /////打印日志
        }
    }
    void SetTcpNoDelay(bool a_on)
    {
        int opval = a_on ? 1 : 0;
        ::setsockopt(m_sock_fd_, IPPROTO_TCP, TCP_NODELAY, &opval, sizeof(opval));
    }
    void SetReuseAddr(bool a_on)
    {
        int opval = a_on ? 1 : 0;
        ::setsockopt(m_sock_fd_, SOL_SOCKET, SO_REUSEADDR, &opval, sizeof(opval));
    }
    void SetReusePort(bool a_on)
    {
        int opval = a_on ? 1 : 0;
        ::setsockopt(m_sock_fd_, SOL_SOCKET, SO_REUSEPORT, &opval, sizeof(opval));
    }
    void SetKeepAlive(bool a_on)
    {
        int opval = a_on ? 1 : 0;
        ::setsockopt(m_sock_fd_, SOL_SOCKET, SO_KEEPALIVE, &opval, sizeof(opval));
    }
    static void SetNoBlockAndCloseOnExec(int a_sock_fd)
    {
        int flags = ::fcntl(a_sock_fd, F_GETFL, 0);
        flags |= O_NONBLOCK;
        ::fcntl(a_sock_fd, F_SETFL, flags);

        flags = ::fcntl(a_sock_fd, F_GETFD, 0);
        flags |= FD_CLOEXEC;
        ::fcntl(a_sock_fd, F_SETFD, flags);
    }
    static void ExtractIPAndPortFromSockAddr(struct sockaddr* a_addr, std::string& a_ip, unsigned short& a_port)
    {
        struct sockaddr_in* addr_in = (struct sockaddr_in*)&a_addr;
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
            return ret;
        }
        ExtractIPAndPortFromSockAddr(&addr, a_ip, a_port);
        return 0;
    }
    static int Read(int a_fd, void* a_buf, size_t a_len)
    {   
        return ::read(a_fd, a_buf, a_len);
    }
    static int Readv(int a_fd, const struct iovec* a_iov, int a_iovcnt)
    {
        return ::readv(a_fd, a_iov, a_iovcnt);
    }
    static int Write(int a_fd, const void* a_buf, size_t a_len)
    {
        return ::write(a_fd, a_buf, a_len);
    }
};
}

#endif