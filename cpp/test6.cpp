#include <unistd.h>
#include <sys/syscall.h>

#include <iostream>
#include <cstdlib>
#include <cmath>
#include "./toolbox/string_function.hpp"
#include "./toolbox/time_function.hpp"
#include "./toolbox/uuid.hpp"

#define MAX_PRO ( 10000.0f )

int main(int argc, char** argv)
{
    // unsigned long long int count = 51240;
    // long long int l_activity_pro = 2500;
    // unsigned int max_rate = 15000;
    // unsigned long long int  act = (MAX_PRO + l_activity_pro);
    // float l_cur_val = count * act/MAX_PRO;
    // double radius_base = 1000.0, radius_step = 1500.0;
    // int count = 0, point_total = 500, layer_count = 3;
    // double xc = (double)0.0, yc = (double)0.0;;
    // double tx = 0.0, ty = 0.0;
    // long long int tz = 0;
    // double radian = 0.0, radius = 0.0, radian_step = 0.0;
    // for (int i = 0; i < 17; i++) ///层
    // {
    //     radius = radius_base + radius_step * double(i);
    //     layer_count += (i + 1);
    //     radian = 2*M_PI/double(layer_count);
    //     for(int j = 0; j < layer_count; j++) ///层上点数量
    //     {
    //         radian_step = radian * double(j);
    //         tx = xc + radius * std::cos(radian_step);
    //         ty = yc + radius * std::sin(radian_step);
    //         std::cout << "tx: " << (long int)(tx) << " ty: " << (long int)ty << " radius " << radius << " layer_count " << layer_count
    //             << " i="<<i << std::endl;
    //     }
    // }
    // int tmp_grid_idx = 0;
    // int count = 0;
    // int row_offset = 5, col_offset = 5, row = 0, col = 0;
    // int row_grid_num = 160, col_grid_num = 160;
    // int grid_idx = 8;
    // row = grid_idx / row_grid_num;
    // col = grid_idx % col_grid_num;
    // for (int i = -row_offset; i <= row_offset; ++i)//行
    // {
    //     if (row+i >= 0 && row+i < row_grid_num)
    //     {
    //         for (int j = -col_offset; j <= col_offset; ++j)//列
    //         {
    //             if (col+j >= 0 && col+j < col_grid_num)
    //             {
    //                 if ((i == -row_offset || i == row_offset) && (j == -col_offset || j == col_offset)) continue;
    //                 tmp_grid_idx = grid_idx + i*row_grid_num + j;
    //                 // if (tmp_grid_idx < 0 || tmp_grid_idx > 8) continue;
    //                 count ++;
    //                 std::cout << "i: " <<i<< " j; " <<j<< " tmp_grid_idx " << tmp_grid_idx<< std::endl;
    //             }
    //         }
    //     }
    // }
    int view_offset_ = 6;
    int abs_i = 0, abs_j = 0;
    for (int i = -view_offset_; i <= view_offset_; ++i)//行
        for (int j = -view_offset_; j <= view_offset_; ++j)//列
        {
            // std::cout << "i: " <<i<< " j; " <<j<< std::endl;
            abs_i = std::abs(i);
            abs_j = std::abs(j);
            if (abs_i > 2 && abs_j == 6)
            {
                std::cout << "i: " <<i<< " j; " <<j<< std::endl;
            }
            else if(abs_j > 2 && abs_i == 6)
            {
                std::cout << "i: " <<i<< " j; " <<j<< std::endl;
            }
            else if (abs_j > 3 && abs_i == 5)
            {
                std::cout << "i: " <<i<< " j; " <<j<< std::endl;
            }
            else if (abs_i > 3 && abs_j == 5)
            {
                std::cout << "i: " <<i<< " j; " <<j<< std::endl;
            }
        }
    // std::cout<< " count " << count<< std::endl;
    std::string r = su::replace_all_fast("shs;djfspodjfsdf", "dj", "<jjjjjjjj>");
    std::cout <<" rrr=" << r << std::endl;

    std::set<int> s1{3,84,4,93,9,32,84,32};
    std::string s1_str = su::Container2String(s1);
    std::map<int, int> m1{{3,84},{4,93},{9,32},{32,84}};
    std::string m1_str = su::Container2String(m1);
    std::cout << "s1_str=" << s1_str << " m1_str="<< m1_str<< std::endl;
    unsigned long long int now_ms = su::MicroTimeCR();
    std::cout << "now_ms=" << now_ms << std::endl;
    std::cout << "DateToTimeStamp=" << su::DateToTimeStamp(20250311) << std::endl;
    std::cout << "DateYearMonthDayString=" << su::DateYearMonthDayString(now_ms/1000000) << std::endl;
    std::cout << "DateYearMonthString=" << su::DateYearMonthString(now_ms/1000000) << std::endl;
    uint64_t uuid = su::sp_uuid();
    std::cout << "uuid=" << uuid << std::endl;
    return 0;
}