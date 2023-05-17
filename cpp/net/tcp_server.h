/*
 * @Copyright: 
 * @file name: File name
 * @Data: Do not edit
 * @LastEditor: 
 * @LastData: 
 * @Describe: 
 */
#ifndef _TCP_SERVER_H_
#define _TCP_SERVER_H_

#include "../toolbox/original_dependence.hpp"

namespace su
{

class tcp_server: public Noncopyable
{
private:
    /* data */
public:
    tcp_server(/* args */);
    ~tcp_server();

    void launch();
};


}
#endif