
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>

#include "tcp_server.h"
#include "../toolbox/string_function.hpp"
#include "../base/acceptor.h"
#include "../base/event_loop.h"
using namespace su;

tcp_server::tcp_server(EventLoop* a_loop, const std::string& a_ip_port):m_ip_port_(a_ip_port),m_acceptor_(a_loop, a_ip_port, true)
{
    m_loop_ = a_loop;
    m_conn_id = 1;
    m_acceptor_->SetNewConnectionCallback(std::bind(NewConnection, this, std::placeholders::_1))
}

tcp_server::~tcp_server()
{
}
/// @brief 服务启动
void tcp_server::Run()
{

}

void tcp_server::NewConnection(int a_fd, std::string a_peer_id, unsigned short a_peer_port)
{

}

void tcp_server::RemoveConnection(unsigned int a_conn_id)
{

}

void tcp_server::SetThreadNum(int a_num)
{

}