#ifndef _SKIP_LIST_HPP_
#define _SKIP_LIST_HPP_

#include "../algorithm/ramdom_disorder.hpp"
#include <cstdio>
#include <vector>

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
    unsigned long m_length_;
    int m_layer_level_;
public:
    Skiplist();
    ~Skiplist();

public:
    SkiplistNode* Insert(unsigned int a_value, double a_score);
    SkiplistNode* Find(double a_score);
    SkiplistNode* Erase(double a_score);

    // Zset需要的新接口
    unsigned long length() const { return m_length_; }
    long Rank(double a_score) const;
    std::vector<SkiplistNode*> RangeByScore(double min_score, double max_score) const;

    void dump() const;
};

Skiplist::Skiplist()
{
    m_head_ = new SkiplistNode();
    m_head_->value = 0;
    m_head_->score = 0;
    m_head_->prev = nullptr;
    for (int i = 0; i < MAX_LEVEL; ++i) {
        m_head_->level_arr[i].next = nullptr;
        m_head_->level_arr[i].span = 0;
    }
    m_length_ = 0;
    m_layer_level_ = 1;
}

Skiplist::~Skiplist()
{
    SkiplistNode* cur = m_head_->level_arr[0].next;
    while (cur) {
        SkiplistNode* next = cur->level_arr[0].next;
        delete cur;
        cur = next;
    }
    delete m_head_;
}

SkiplistNode* Skiplist::Insert(unsigned int a_value, double a_score)
{
    SkiplistNode* update[MAX_LEVEL];
    unsigned int rank[MAX_LEVEL];

    SkiplistNode* cur = m_head_;
    for (int i = m_layer_level_ - 1; i >= 0; --i) {
        rank[i] = (i == m_layer_level_ - 1) ? 0 : rank[i + 1];
        while (cur->level_arr[i].next &&
               cur->level_arr[i].next->score < a_score) {
            rank[i] += cur->level_arr[i].span;
            cur = cur->level_arr[i].next;
        }
        update[i] = cur;
    }

    int lev = GetRandomLevel();
    if (lev > m_layer_level_) {
        for (int i = m_layer_level_; i < lev; ++i) {
            rank[i] = 0;
            update[i] = m_head_;
            update[i]->level_arr[i].span = m_length_;
        }
        m_layer_level_ = lev;
    }

    SkiplistNode* node = new SkiplistNode();
    node->value = a_value;
    node->score = a_score;
    node->prev = nullptr;

    for (int i = 0; i < lev; ++i) {
        node->level_arr[i].next = update[i]->level_arr[i].next;
        update[i]->level_arr[i].next = node;
        node->level_arr[i].span = update[i]->level_arr[i].span - (rank[0] - rank[i]);
        update[i]->level_arr[i].span = (rank[0] - rank[i]) + 1;
    }
    for (int i = lev; i < m_layer_level_; ++i) {
        update[i]->level_arr[i].span++;
    }

    node->prev = (update[0] == m_head_) ? nullptr : update[0];
    if (node->level_arr[0].next)
        node->level_arr[0].next->prev = node;

    for (int i = lev; i < MAX_LEVEL; ++i) {
        node->level_arr[i].next = nullptr;
        node->level_arr[i].span = 0;
    }

    ++m_length_;
    return node;
}

SkiplistNode* Skiplist::Find(double a_score)
{
    SkiplistNode* cur = m_head_;
    for (int i = m_layer_level_ - 1; i >= 0; --i) {
        while (cur->level_arr[i].next &&
               cur->level_arr[i].next->score < a_score) {
            cur = cur->level_arr[i].next;
        }
    }
    cur = cur->level_arr[0].next;
    if (cur && cur->score == a_score)
        return cur;
    return nullptr;
}

SkiplistNode* Skiplist::Erase(double a_score)
{
    SkiplistNode* update[MAX_LEVEL];
    SkiplistNode* cur = m_head_;
    for (int i = m_layer_level_ - 1; i >= 0; --i) {
        while (cur->level_arr[i].next &&
               cur->level_arr[i].next->score < a_score) {
            cur = cur->level_arr[i].next;
        }
        update[i] = cur;
    }

    SkiplistNode* target = cur->level_arr[0].next;
    if (!target || target->score != a_score)
        return nullptr;

    for (int i = 0; i < m_layer_level_; ++i) {
        if (update[i]->level_arr[i].next != target) {
            update[i]->level_arr[i].span--;
        } else {
            update[i]->level_arr[i].span += target->level_arr[i].span - 1;
            update[i]->level_arr[i].next = target->level_arr[i].next;
        }
    }

    if (target->level_arr[0].next)
        target->level_arr[0].next->prev = target->prev;

    while (m_layer_level_ > 1 &&
           m_head_->level_arr[m_layer_level_ - 1].next == nullptr) {
        --m_layer_level_;
    }

    --m_length_;
    return target;
}

long Skiplist::Rank(double a_score) const
{
    long rank = 0;
    SkiplistNode* cur = m_head_;
    for (int i = m_layer_level_ - 1; i >= 0; --i) {
        while (cur->level_arr[i].next &&
               cur->level_arr[i].next->score < a_score) {
            rank += cur->level_arr[i].span;
            cur = cur->level_arr[i].next;
        }
    }
    cur = cur->level_arr[0].next;
    if (cur && cur->score == a_score) {
        return rank + 1; // 1-based rank
    }
    return 0; // Not found
}

std::vector<SkiplistNode*> Skiplist::RangeByScore(double min_score, double max_score) const
{
    std::vector<SkiplistNode*> result;
    SkiplistNode* cur = m_head_;
    for (int i = m_layer_level_ - 1; i >= 0; --i) {
        while (cur->level_arr[i].next &&
               cur->level_arr[i].next->score < min_score) {
            cur = cur->level_arr[i].next;
        }
    }
    cur = cur->level_arr[0].next;
    while (cur && cur->score <= max_score) {
        result.push_back(cur);
        cur = cur->level_arr[0].next;
    }
    return result;
}

void Skiplist::dump() const
{
    for (int i = m_layer_level_ - 1; i >= 0; --i) {
        printf("level %d: ", i);
        SkiplistNode* cur = m_head_->level_arr[i].next;
        while (cur) {
            printf("[val=%u score=%.2f span=%u] -> ", cur->value, cur->score, cur->level_arr[i].span);
            cur = cur->level_arr[i].next;
        }
        printf("NULL\n");
    }
}

#endif