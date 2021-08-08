#ifndef _SKIP_LIST_HPP_
#define _SKIP_LIST_HPP_


struct SkiplistNode
{
    //robj* obj;
    double score;
    struct SkiplistNode* next;
    struct SkiplistLevel {
        struct SkiplistNode* pre;
        unsigned int span;
    } level[];
};

struct Skiplist
{
    struct SkiplistNode* head, *tail;
    unsigned long length;
    int level;
public:
    Skiplist();
    ~Skiplist();

public:
    SkiplistNode* insert();
    SkiplistNode* find();
    SkiplistNode* erase();

    void dump();
};


#endif