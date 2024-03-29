/*
 * @Copyright: 
 * @file name: File name
 * @Data: Do not edit
 * @LastEditor: 
 * @LastData: 
 * @Describe: 
 */
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/epoll.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <errno.h>
#include "tcp_server.h"
#include "../toolbox/string_function.hpp"

using namespace su;

tcp_server::tcp_server(const std::string& a_ip_port):m_ip_port_(a_ip_port)
{
    m_conn_id = 1;
    m_ltn_ip_="";
    m_ltn_port_=0;
}

tcp_server::~tcp_server()
{
}
/// @brief 服务启动
void tcp_server::Launch()
{
    std::vector<std::string> str_vec;
    su::Split(m_ip_port, ":", str_vec);
    if (str_vec.size() == 1)
    {
        m_ltn_port_ = su::String2Number<int>(str_vec[0]);
    }
    else if (str_vec.szie() == 2)
    {
        m_ltn_ip_ = str_vec[0];
        m_ltn_port_ = su::String2Number<int>(str_vec[1]);
    }
    else
    {
        //log
        return
    }
    int listen_fd=0;
    int ret = 0;
    listen_fd = ::socket(PF_INET, SOCK_STREAM, 0);
    if (listen_fd < 0)
    {
        ///log
        return;
    }
    struct sockaddr_in my_addr;
    memset(&my_addr, 0, sizeof(my_addr));
    my_addr.sin_family = AF_INET;
    my_addr.sin_port = ::htons(m_ltn_port_);
    if (m_ltn_ip_=="")
        my_addr.sin_addr.s_addr = ::htonl(INADDR_ANY);
    else
        my_addr.sin_addr.s_addr = ::inet_addr(m_ltn_ip_.c_str());
    ret = ::bind(listen_fd, (struct sockaddr*)&my_addr, sizeof(my_addr));
    if (ret < 0)
    {
        ///log
        return
    }
    int ep_fd = epoll_create(100000);
    struct epoll_event ep_event, ep_events[10000];
    ep_event.events = EPOLLIN | EPOLLET;
    ep_event.data.fd = listen_fd;
    ret = epoll_ctl(ep_fd, EPOLL_CTL_ADD, listen_fd, &ep_event);
    ret = ::listen(listen_fd, SOMAXCONN);
    if (ret < 0)
    {
        //log
        return;
    }
}