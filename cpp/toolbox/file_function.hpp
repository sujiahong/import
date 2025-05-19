////////////
/////文件操作函数
///////////

#ifndef __FILE_FUNCTION_HPP__
#define __FILE_FUNCTION_HPP__

#include <sys/types.h>
#include <sys/stat.h>
#include <unistd.h>
#include <fcntl.h>
#include <string>
#include <sys/mman.h>

namespace su
{

char* FileOpenWithMMap(std::string a_name, int a_flags, int a_mode=0)
{
    int fd = 0;
    if (a_mode > 0)
        fd = ::open(a_name.c_str(), a_flags, a_mode);
    else
        fd = ::open(a_name.c_str(), a_flags, 0644);
    if (fd < 0)
        throw std::runtime_error("open file error "+ a_name);
    int len = 4096;
    // lseek(fd, len-1, SEEK_END);
    // write(fd, "", len);
    ftruncate(fd, len);
    char* maddr = (char*)mmap(NULL, len, PROT_WRITE|PROT_READ, MAP_SHARED, fd, 0);
    //std::cout <<" mmap  maddr="<<(void*)maddr<<std::endl;
    if (maddr == MAP_FAILED)
    {
        close(fd);
        return 0;
    }
    close(fd);
    return maddr;
}

int FileRead()
{

}

int FileWrite()
{

}

}


#endif