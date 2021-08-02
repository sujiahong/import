#ifndef _SKIP_LIST_HPP_
#define _SKIP_LIST_HPP_


struct SkiplistNode
{
    robj* obj;
    double score;
    struct SkiplistNode*backward;
    struct SkiplistLevel {
        struct SkiplistNode* forward;
        unsigned int span;
    } level[];
};

struct Skiplist
{
    struct SkiplistNode* head, *tail;
    unsigned long length;
    int level;
};


#endif