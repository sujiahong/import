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
#include <errno.h>
#include <string>

void main()
{
    int sock_fd;
    int ret = 0;
    sock_fd = socket(PF_INET, SOCK_STREAM, 0);

    struct sockaddr_in my_addr;
    memset(&my_addr, 0, sizeof(my_addr));
    my_addr.sin_family = AF_INET;
    my_addr.sin_port = htons(5679);
    my_addr.sin_addr.s_addr = htonl(INADDR_ANY); 

    ret = bind(sock_fd, (struct sockaddr*)&my_addr, sizeof(my_addr));
    if (ret < 0)
    {

    }
    int ep_fd = epoll_create(100000);
    struct epoll_event ep_event, ep_events[10000];
    ep_event.events = EPOLLIN | EPOLLET;
    ep_event.data.fd = sock_fd;
    ret = epoll_ctl(ep_fd, EPOLL_CTL_ADD, sock_fd, &ep_event);
    ret = listen(sock_fd, SOMAXCONN);
    if (ret < 0)
    {

    }
    struct sockaddr_in peer_addr;
    socklen_t peer_addr_len = sizeof(peer_addr);
    while(1)
    {
        ret = epoll_wait(ep_fd, ep_events, 10000, -1);
        if (ret >= 0)
        {
            for (int i = 0; i < ret; ++i)
            {
                if (ep_events[i].events & EPOLLIN)//读
                {
                    if (ep_events[i].data.fd == sock_fd)
                    {
                        int fd = accept(sock_fd, (struct sockaddr*)&peer_addr, &peer_addr_len);/////fd可以发送，接收数据了
                        if (fd != -1)
                        {
                        
                        }
                        ep_event.events = EPOLLIN;
                        ep_event.data.fd = fd;
                        ret = epoll_ctl(ep_fd, EPOLL_CTL_ADD, fd, &ep_event);
                        ////创建处理sock数据任务
                    }
                    else
                    {

                    }
                }
                else if (ep_events[i].events & EPOLLOUT)//写
                {

                }
            }
        }
        else
        {
            continue;
        }
    }
}
