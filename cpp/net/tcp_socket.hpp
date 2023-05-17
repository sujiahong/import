/*
 * @Copyright: 
 * @file name: File name
 * @Data: Do not edit
 * @LastEditor: 
 * @LastData: 
 * @Describe: 
 */
/*
 * @Copyright: 
 * @file name: File name
 * @Data: Do not edit
 * @LastEditor: 
 * @LastData: 
 * @Describe: 
 */

#ifndef _SOCKET_HPP_
#define _SOCKET_HPP_

#include <string>

class TcpSocket
{
private:

public:
    TcpSocket(std::string a_ip, unsigned int a_port);
    ~TcpSocket();

    int CreateServerSocket(std::string a_ip, unsigned int a_port);
    int CreateClientSocket(std::string a_ip, unsigned int a_port);

    int Accepter(int a_fid, std::string& a_ip, unsigned int& a_port);

    int SetSocketOption(int a_fid, int a_opt);
};                                          

#endif