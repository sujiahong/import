
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include "../base/socket.h"
#include "tcp_server.h"
#include "../toolbox/string_function.hpp"

using namespace su;

tcp_server::tcp_server(const std::string& a_ip_port):m_ip_port_(a_ip_port)
{
    m_conn_id = 1;
    std::vector<std::string> str_vec;
    su::Split(m_ip_port_, ":", str_vec);
    if (str_vec.size() == 1)
    {
        m_ltn_ip_="";
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
    }
}

tcp_server::~tcp_server()
{
}
/// @brief 服务启动
void tcp_server::Launch()
{
    listen_sock = Socket(PF_INET, SOCK_STREAM);
    if (m_ltn_ip_ == "")
        listen_sock.Bind(m_ltn_port_);
    else
        listen_sock.Bind(m_ltn_ip_, m_ltn_port_);
    listen_sock.Listen();

    
}