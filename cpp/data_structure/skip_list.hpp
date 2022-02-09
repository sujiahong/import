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
    struct SkiplistNode* m_head_;
    unsigned long m_length_;    ////链表长度
    int m_layer_level_;        /////层数
public:
    Skiplist();
    ~Skiplist();

public:
    SkiplistNode* Insert(unsigned int a_value, double a_score);
    SkiplistNode* Find(double a_score);
    SkiplistNode* Erase(double a_score);

    void dump();
};

Skiplist::Skiplist()
{
    m_head_ = new SkiplistNode();
    m_length_ = 0;
    m_layer_level_ = 0;
}

Skiplist::~Skiplist()
{

}


SkiplistNode* Skiplist::Insert(unsigned int a_value, double a_score)
{
    return 0;
}

SkiplistNode* Skiplist::Find(double a_score)
{
    return 0;
}

SkiplistNode* Skiplist::Erase(double a_score)
{
    return 0;
}

#endif