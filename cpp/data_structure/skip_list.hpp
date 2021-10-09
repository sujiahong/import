#ifndef _SKIP_LIST_HPP_
#define _SKIP_LIST_HPP_

struct RObj
{

};

struct SkiplistNode
{
    RObj* obj;
    double score;
    struct SkiplistNode*backward;
    struct SkiplistLevel {
        struct SkiplistNode* forward;
        unsigned int span;
    } level[];
};

class Skiplist
{
    struct SkiplistNode* head, *tail;
    unsigned long length;
    int level;
public:
    Skiplist();
    ~Skiplist();

public:
    void Insert();
    void Delete();

    void Query();

};


#endif