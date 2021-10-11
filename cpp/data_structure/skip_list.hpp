#ifndef _SKIP_LIST_HPP_
#define _SKIP_LIST_HPP_

#include "../algorithm/ramdom_disorder.hpp"

#define MAX_LEVEL 8

struct SkiplistNode
{
    unsigned int value;
    double score;
    struct SkiplistNode* prev;
    struct SkiplistLevel {
        struct SkiplistNode* next;
        unsigned int span;
    } level_arr[MAX_LEVEL];
};

int GetRandomLevel()
{
    int lev = 1;
    while((RangeRandom(1, 10000) & 1) == 0)
    {
        ++lev;
    }
    return lev < MAX_LEVEL ? lev : MAX_LEVEL;
}

class Skiplist
{
    struct SkiplistNode* head;
    unsigned long length;
    int level;
public:
    Skiplist();
    ~Skiplist();

public:
    SkiplistNode* Insert(unsigned int a_value, double a_score);
    SkiplistNode* Find(double a_score);
    SkiplistNode* Erase(double a_score);

    void dump();
};


SkiplistNode* Skiplist::Insert(unsigned int a_value, double a_score)
{

}

SkiplistNode* Skiplist::Find(double a_score)
{

}

SkiplistNode* Skiplist::Erase(double a_score)
{

}

#endif