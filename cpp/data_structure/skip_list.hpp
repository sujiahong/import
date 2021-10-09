#ifndef _SKIP_LIST_HPP_
#define _SKIP_LIST_HPP_

#include "../algorithm/ramdom_disorder.hpp"

#define MAX_LEVEL 5

struct RObj
{
    
};

struct SkiplistNode
{
    unsigned int value;
    double score;
    struct SkiplistNode* next;
    struct SkiplistLevel {
        struct SkiplistNode* pre;
        unsigned int span;
    } level[];
};

int GetRandomLevel()
{
    int lev = 1;
    while((RangeRandom(1, 10000) & 1) == 0)
    {
        ++lev;
    }
    return lev > MAX_LEVEL ? MAX_LEVEL : lev;
}

class Skiplist
{
    struct SkiplistNode* head, *tail;
    unsigned long length;
    int level;
public:
    Skiplist();
    ~Skiplist();

public:
    SkiplistNode* insert(unsigned int a_value, double a_score);
    SkiplistNode* find(unsigned int a_value);
    SkiplistNode* erase(unsigned int a_value);

    void dump();
};


#endif