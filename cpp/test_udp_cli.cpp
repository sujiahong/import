#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <string.h>
#include <arpa/inet.h>
#include <errno.h>

char* SERVERIP = "127.0.0.1";

int main()
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
    servaddr.sin_addr.s_addr = inet_addr(SERVERIP);

    int ret;
    char sendbuf[1024]={0};
    char recvbuf[1024]={0};

    while(fgets(sendbuf, sizeof(sendbuf), stdin) != NULL)
    {
        printf("向服务发送：%s\n", sendbuf);
        sendto(sock_fd, sendbuf, strlen(sendbuf), 0, (struct sockaddr*)&servaddr, sizeof(servaddr));
        ret = recvfrom(sock_fd, recvbuf, sizeof(recvbuf), 0, NULL, NULL);
        if (ret <= 0)
        {
            printf("接收失败：%d\n", errno);
            if (errno == EINTR)
                continue;
        }
        printf("从服务器接收：%s\n", recvbuf);
        memset(sendbuf, 0, sizeof(sendbuf));
        memset(recvbuf, 0, sizeof(recvbuf));
    }
    close(sock_fd);
    return 0;
}