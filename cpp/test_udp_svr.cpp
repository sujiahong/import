#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <string.h>
#include <errno.h>

int main(void)
{
    int sock_fd;
    if ((sock_fd = socket(PF_INET, SOCK_DGRAM, 0)) < 0)
    {
        printf("创建socket失败 %d\n", sock_fd);
        return -1;
    }
    struct sockaddr_in servaddr;
    memset(&servaddr, 0, sizeof(servaddr));
    servaddr.sin_family = AF_INET;
    servaddr.sin_port = htons(8887);
    servaddr.sin_addr.s_addr = htonl(INADDR_ANY);
    printf("开始绑定ip地址");
    if (bind(sock_fd, (struct sockaddr*)&servaddr, sizeof(servaddr)) < 0)
    {
        printf("绑定ip地址失败");
        return -2;
    }
    char recvbuf[1024] = {0};
    struct sockaddr_in peeraddr;
    socklen_t peerlen;
    int n;
    peerlen = sizeof(peeraddr);
    while(1)
    {
        memset(recvbuf, 0, sizeof(recvbuf));
        n = recvfrom(sock_fd, recvbuf, sizeof(recvbuf), 0, (struct sockaddr*)&peeraddr, &peerlen);
        if (n <= 0)
        {
            if (errno == EINTR)
                continue;
        }
        else if (n > 0)
        {
            printf("接收的数据：%s\n", recvbuf);
            sendto(sock_fd, recvbuf, n, 0, (struct sockaddr*)&peeraddr, peerlen);
            printf("回送的数据：%s\n", recvbuf);
        }
    }
    close(sock_fd);
    return 0;
}