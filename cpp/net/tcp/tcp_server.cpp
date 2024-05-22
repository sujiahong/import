
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>

#include "tcp_server.h"
#include "../../toolbox/string_function.hpp"
#include "../base/acceptor.h"
#include "../base/event_loop.hpp"
#include "../base/connection.h"
#include "../base/socket.h"
using namespace su;

TcpServer::TcpServer(EventLoop* a_loop, const std::string& a_ip_port, const std::string& a_name):
    m_name_(a_name),
    m_ip_port_(a_ip_port),
    m_accept_loop_(a_loop),
    m_acceptor_(a_loop, a_ip_port, true),
    m_thread_pool_(a_loop, 4)
{
    m_conn_id_ = 1;
    m_acceptor_.SetNewConnectionCallback(std::bind(NewConnection, this, std::placeholders::_1));
}

TcpServer::~TcpServer()
{
}
/// @brief 服务启动
void TcpServer::Run()
{
    m_thread_pool_.Start();
}

void TcpServer::NewConnection(int a_fd, const std::string& a_peer_ip, unsigned short a_peer_port)
{
    if (a_fd == -1) {
        ////todo log
        return;
    }
    EventLoop* ioloop = m_thread_pool_.GetNextLoop();
    std::string connName = m_name_ + "-" + m_ip_port_ + "#" + std::to_string(m_conn_id_);
    m_conn_id_++;
    std::string local_ip;
    unsigned short local_port;
    Socket::GetSelfIPAndPort(a_fd, local_ip, local_port);
    TCP_CONNECTION_PTR conn(new Connection(ioloop, a_fd, connName, local_ip, local_port, a_peer_ip, a_peer_port));
    m_connections_[a_fd] = conn;
    conn->SetConnectionCallback(m_connection_callback_);
    conn->SetMessageCallback(m_message_callback_);
    conn->SetWriteCompleteCallback(m_write_complete_callback_);
    conn->SetCloseCallback(
        std::bind(&TcpServer::RemoveConnection, this, std::placeholders::_1)); // FIXME: unsafe 数据线程调过来的
}

void TcpServer::RemoveConnection(int a_fd)
{
    m_connections_.erase(a_fd);
}

void TcpServer::SetThreadNum(int a_num)
{
    m_thread_pool_.SetThreadNum(a_num);
}