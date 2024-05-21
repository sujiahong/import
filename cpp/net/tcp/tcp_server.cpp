
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>

#include "tcp_server.h"
#include "../toolbox/string_function.hpp"
#include "../base/acceptor.h"
#include "../base/event_loop.h"
using namespace su;

TcpServer::TcpServer(EventLoop* a_loop, const std::string& a_ip_port):m_ip_port_(a_ip_port),m_acceptor_(a_loop, a_ip_port, true)
{
    m_accept_loop_ = a_loop;
    m_conn_id = 1;
    m_acceptor_.SetNewConnectionCallback(std::bind(NewConnection, this, std::placeholders::_1));
}

TcpServer::~TcpServer()
{
}
/// @brief 服务启动
void TcpServer::Run()
{

}

void TcpServer::NewConnection(int a_fd, std::string a_peer_id, unsigned short a_peer_port)
{

}

void TcpServer::RemoveConnection(unsigned int a_conn_id)
{

}

void TcpServer::SetThreadNum(int a_num)
{

}