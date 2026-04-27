#ifndef _REDIS_ZSET_HPP_
#define _REDIS_ZSET_HPP_

#include "skip_list.hpp"
#include <unordered_map>
#include <string>
#include <vector>
#include <cstdio>
#include <functional>

// O(1) score mapping
struct Dict
{
    std::unordered_map<std::string, double> data;

    bool insert(const std::string& member, double score) {
        return data.emplace(member, score).second;
    }
    bool update(const std::string& member, double score) {
        auto it = data.find(member);
        if (it == data.end()) return false;
        it->second = score;
        return true;
    }
    bool erase(const std::string& member) {
        return data.erase(member) > 0;
    }
    bool find(const std::string& member, double& score) const {
        auto it = data.find(member);
        if (it == data.end()) return false;
        score = it->second;
        return true;
    }
};

struct ZsetEntry
{
    std::string member;
    double score;
};

class Zset
{
private:
    Dict m_dict_;
    Skiplist m_skiplist_;

    // Store original member strings mapped from their hash value
    // (since SkiplistNode only supports unsigned int value)
    std::unordered_map<unsigned int, std::string> m_hash_to_member_;

public:
    // Add or update member. Returns true if added, false if updated.
    bool zadd(const std::string& member, double score);

    // Remove member. Returns true if existed.
    bool zrem(const std::string& member);

    // Get score of member. Returns true if exists.
    bool zscore(const std::string& member, double& score) const;

    // Get 0-based rank of member ordered by score. Returns -1 if not exists.
    long zrank(const std::string& member) const;

    // Get members with scores in [min_score, max_score]
    std::vector<ZsetEntry> zrangebyscore(double min_score, double max_score) const;

    // Total elements
    unsigned long zcard() const { return m_skiplist_.length(); }

    void dump() const { m_skiplist_.dump(); }
};

bool Zset::zadd(const std::string& member, double score)
{
    unsigned int hash_val = static_cast<unsigned int>(std::hash<std::string>{}(member));
    m_hash_to_member_[hash_val] = member;

    double old_score;
    if (m_dict_.find(member, old_score)) {
        if (old_score == score) return false;

        // Remove old score from skiplist
        SkiplistNode* erased = m_skiplist_.Erase(old_score);
        delete erased;

        // Insert new score
        m_dict_.update(member, score);
        m_skiplist_.Insert(hash_val, score);
        return false; // updated
    }

    m_dict_.insert(member, score);
    m_skiplist_.Insert(hash_val, score);
    return true; // added
}

bool Zset::zrem(const std::string& member)
{
    double score;
    if (!m_dict_.find(member, score)) return false;

    m_dict_.erase(member);
    unsigned int hash_val = static_cast<unsigned int>(std::hash<std::string>{}(member));
    m_hash_to_member_.erase(hash_val);

    SkiplistNode* erased = m_skiplist_.Erase(score);
    if (erased) delete erased;

    return true;
}

bool Zset::zscore(const std::string& member, double& score) const
{
    return m_dict_.find(member, score);
}

long Zset::zrank(const std::string& member) const
{
    double score;
    if (!m_dict_.find(member, score)) return -1;
    long rank = m_skiplist_.Rank(score);
    return rank > 0 ? rank - 1 : -1; // Convert 1-based rank to 0-based
}

std::vector<ZsetEntry> Zset::zrangebyscore(double min_score, double max_score) const
{
    std::vector<ZsetEntry> result;
    std::vector<SkiplistNode*> nodes = m_skiplist_.RangeByScore(min_score, max_score);

    for (SkiplistNode* node : nodes) {
        auto it = m_hash_to_member_.find(node->value);
        if (it != m_hash_to_member_.end()) {
            result.push_back({it->second, node->score});
        }
    }
    return result;
}

#endif