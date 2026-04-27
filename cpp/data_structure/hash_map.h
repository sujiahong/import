#ifndef _HASH_MAP_H_
#define _HASH_MAP_H_ 

#include <map>
#include <cassert>
#include "list_head.h"
#include "../base/base_macro.h"
#include "../base/original_base.h"

namespace su
{
//////碰撞节点数量多可以用map存储，大于4个。
template<typename KeyT, typename ValueT, unsigned int NumBuckets> 
class HashMap: public Noncopyable
{
private: ////类型定义
    struct Pair
    {
        KeyT key;
        ValueT value;
        inline Pair(const KeyT& k, const ValueT& v):key(k),value(v)  ///存两个值
        {}
        inline Pair(const KeyT& k):key(k)  ////存一个值
        {}
    };
    struct HashEntity
    {
        _double_list_link itor_link_;
        _double_list_link bucket_link_;
        Pair pair;

        inline HashEntity(const KeyT& k, const ValueT& v):pair(k,v)
        {
            INIT_HEAD(&itor_link_);
            INIT_HEAD(&bucket_link_);
        }
        inline HashEntity(const KeyT& k):pair(k)
        {
            INIT_HEAD(&itor_link_);
            INIT_HEAD(&bucket_link_);
        }
    };
public: ////类型定义
    typedef HashMap<KeyT, ValueT, NumBuckets> _SelfType;
    class _Iterator
    {
    protected:
        _SelfType* itor_master_;
        HashEntity* entity_;
        friend class HashMap<KeyT, ValueT, NumBuckets>;
    public:
        _Iterator(_SelfType* im=0, HashEntity* entity=0):itor_master_(im), entity_(entity)
        {}
        inline Pair* operator*() const 
        {
            return &entity_->pair;
        }
        inline Pair* operator->() const 
        {
            return &entity_->pair;
        }
        inline bool operator ==(const _Iterator& r)
        {
            return (itor_master_ == r.itor_master_) && (entity_ == r.entity_);
        }
        inline bool operator !=(const _Iterator& r)
        {
            return (itor_master_ != r.itor_master_) || (entity_ != r.entity_);
        }
        _Iterator operator++()
        {
            if (entity_)
            {
                if (entity_->itor_link_.next_ptr)
                {
                    entity_ = containerof_new(entity_->itor_link_.next_ptr, struct HashEntity, itor_link_);
                }
                else
                    entity_ = 0;
            }
            return *this;
        }
        _Iterator operator++(int)
        {
            _Iterator it(*this);
            if (entity_)
            {
                if (entity_->itor_link_.next_ptr)
                {
                    entity_ = containerof_new(entity_->itor_link_.next_ptr, struct HashEntity, itor_link_);
                }
                else
                    entity_ = 0;
            }
            return it;
        }
        inline _Iterator operator--()
        {
            if (entity_ == 0)////在末尾
            {
                if (itor_master_->itor_root_.next_ptr)
                {
                    entity_ = containerof_new(itor_master_->itor_root_.next_ptr, struct HashEntity, itor_link_);
                } 
            }
            else
            {
                if (&entity_->itor_link_ != itor_master_->itor_root_.prev_ptr)
                {
                    entity_ = containerof_new(entity_->itor_link_.prev_ptr, struct HashEntity, itor_link_);
                }
                else
                    entity_ = 0;
            }
            return *this;
        }
        inline _Iterator operator--(int)
        {
            _Iterator it(*this);
            if (entity_ == 0)////在末尾
            {
                if (itor_master_->itor_root_.next_ptr)
                {
                    entity_ = containerof_new(itor_master_->itor_root_.next_ptr, struct HashEntity, itor_link_);
                } 
            }
            else
            {
                if (&entity_->itor_link_ != itor_master_->itor_root_.prev_ptr)
                {
                    entity_ = containerof_new(entity_->itor_link_.prev_ptr, struct HashEntity, itor_link_);
                }
                else
                    entity_ = 0;
            }
            return it;
        }
    };
    typedef typename _SelfType::_Iterator iterator;
private: ////成员变量
    _double_list_link* table_;
    _double_list_link itor_root_;
    unsigned int total_entity_num_;         //////总数量
    unsigned int table_length_;             //////表长度
public:
    HashMap();
    virtual ~HashMap();

public:
    inline unsigned int size() const{return total_entity_num_};
    inline unsigned int length() const{return table_length_};

    ///清空数据
    void clear();

    iterator find(const KeyT& key);
    iterator insert(const KeyT& key, const ValueT& value);
    //擦除
    int erase(const KeyT& key);
    int erase(iterator iter);
    /* 下标操作 */
    ValueT& operator[] (const KeyT& key);
    ///迭代器
    iterator begin()
    {
        if (itor_root_.prev_ptr)
        {
            return iterator(this, containerof_new(itor_root_.prev_ptr, struct HashEntity, itor_link_));
        }
        return iterator(this);
    }
    iterator end()
    {
        return iterator(this);
    }
protected:
    int init_table(unsigned int num);
};

template<typename KeyT, typename ValueT, unsigned int NumBuckets>
HashMap<KeyT, ValueT, NumBuckets>::HashMap():total_entity_num_(0),table_length_(0)
{
    INIT_HEAD(&itor_root_);
    assert(NumBuckets > 0); ///"NumBuckets must be greater than 0"
    init_table(NumBuckets);
}

template<typename KeyT, typename ValueT, unsigned int NumBuckets>
HashMap<KeyT, ValueT, NumBuckets>::~HashMap()
{
    clear();
    if (table_)
        ::free(table_);
}

const unsigned long _primer_nums_[] = 
{
    2ul, 3ul, 5ul, 7ul, 11ul, 13ul, 17ul, 19ul, 23ul, 29ul, 31ul,
    37ul, 41ul, 43ul, 47ul, 53ul, 59ul, 61ul, 67ul, 71ul, 73ul, 79ul,
    83ul, 89ul, 97ul, 103ul, 109ul, 113ul, 127ul, 137ul, 139ul, 149ul,
    157ul, 167ul, 179ul, 193ul, 199ul, 211ul, 227ul, 241ul, 257ul,
    277ul, 293ul, 313ul, 337ul, 359ul, 383ul, 409ul, 439ul, 467ul,
    503ul, 541ul, 577ul, 619ul, 661ul, 709ul, 761ul, 823ul, 887ul,
    953ul, 1031ul, 1109ul, 1193ul, 1289ul, 1381ul, 1493ul, 1613ul,
    1741ul, 1879ul, 2029ul, 2179ul, 2357ul, 2549ul, 2753ul, 2971ul,
    3209ul, 3469ul, 3739ul, 4027ul, 4349ul, 4703ul, 5087ul, 5503ul,
    5953ul, 6427ul, 6949ul, 7517ul, 8123ul, 8783ul, 9497ul, 10273ul,
    11113ul, 12011ul, 12983ul, 14033ul, 15173ul, 16411ul, 17749ul,
    19183ul, 20753ul, 22447ul, 24281ul, 26267ul, 28411ul, 30727ul,
    33223ul, 35933ul, 38873ul, 42043ul, 45481ul, 49201ul, 53201ul,
    57557ul, 62233ul, 67307ul, 72817ul, 78779ul, 85229ul, 92203ul,
    99733ul, 107897ul, 116731ul, 126271ul, 136607ul, 147793ul,
    159871ul, 172933ul, 187091ul, 202409ul, 218971ul, 236897ul,
    256279ul, 277261ul, 299951ul, 324503ul, 351061ul, 379787ul,
    410857ul, 444487ul, 480881ul, 520241ul, 562841ul, 608903ul,
    658753ul, 712697ul, 771049ul, 834181ul, 902483ul, 976369ul,
    1056323ul, 1142821ul, 1236397ul, 1337629ul, 1447153ul, 1565659ul,
    1693859ul, 1832561ul, 1982627ul, 2144977ul, 2320627ul, 2510653ul,
    2716249ul, 2938679ul, 3179303ul, 3439651ul, 3721303ul, 4026031ul,
    4355707ul, 4712381ul, 5098259ul, 5515729ul, 5967347ul, 6456007ul,
    6984629ul, 7556579ul, 8175383ul, 8844859ul, 9569143ul, 10352717ul,
    11200489ul, 12117689ul, 13109983ul, 14183539ul, 15345007ul,
    16601593ul, 17961079ul, 19431899ul, 21023161ul, 22744717ul,
    24607243ul, 26622317ul, 28802401ul, 31160981ul, 33712729ul,
    36473443ul, 39460231ul, 42691603ul, 46187573ul, 49969847ul,
    54061849ul, 58488943ul, 63278561ul, 68460391ul, 74066549ul,
    80131819ul, 86693767ul, 93793069ul, 101473717ul, 109783337ul,
    118773397ul, 128499677ul, 139022417ul, 150406843ul, 162723577ul,
    176048909ul, 190465427ul, 206062531ul, 222936881ul, 241193053ul,
    260944219ul, 282312799ul, 305431229ul, 330442829ul, 357502601ul,
    386778277ul, 418451333ul, 452718089ul, 489790921ul, 529899637ul,
    573292817ul, 620239453ul, 671030513ul, 725980837ul, 785430967ul,
    849749479ul, 919334987ul, 994618837ul, 1076067617ul, 1164186217ul,
    1259520799ul, 1362662261ul, 1474249943ul, 1594975441ul, 1725587117ul,
    1866894511ul, 2019773507ul, 2185171673ul, 2364114217ul, 2557710269ul,
    2767159799ul, 2993761039ul, 3238918481ul, 3504151727ul, 3791104843ul,
    4101556399ul, 4294967291ul
};
template<typename KeyT, typename ValueT, unsigned int NumBuckets>
int HashMap<KeyT, ValueT, NumBuckets>::init_table(unsigned int num)
{
    unsigned int bsize = 0;
    for (unsigned int i = 0; i < sizeof(_primer_nums_) / sizeof(unsigned long); i++)
    {
        bsize = _primer_nums_[i];
        if (bsize >= num)
            break;
    }
    table_length_ = bsize;
    table_ = (_double_list_link*)malloc(sizeof(_double_list_link) * bsize);
    memset(table_, 0, sizeof(_double_list_link) * bsize);
    return 0;
}

template<typename KeyT, typename ValueT, unsigned int NumBuckets>
void HashMap<KeyT, ValueT, NumBuckets>::clear()
{
    _double_list_link* next_ptr = 0;
    while(next_ptr = dlh_left_pop(&itor_root_))
    {
        HashEntity* entity = containerof_new(next_ptr, struct HashEntity, itor_link_);
        delete entity;
    }
    memset(table_, 0, sizeof(_double_list_link) * table_length_);
    total_entity_num_ = 0;
}

template<typename KeyT, typename ValueT, unsigned int NumBuckets>
typename HashMap<KeyT, ValueT, NumBuckets>::iterator HashMap<KeyT, ValueT, NumBuckets>::find(const KeyT& key)
{
    unsigned int idx = key.hash_value() % table_length_;
    _double_list_link* next = table_[idx].prev_ptr;
    iterator itor(this, 0);
    while(next)
    {
        HashEntity* entity = containerof_new(next, struct HashEntity, bucket_link_);
        if (entity->pair.key == key)
        {
            itor.entity_ = entity;
            return itor;

        }
        next = next->next_ptr;
    }
    return itor;
}
template<typename KeyT, typename ValueT, unsigned int NumBuckets>
typename HashMap<KeyT, ValueT, NumBuckets>::iterator HashMap<KeyT, ValueT, NumBuckets>::insert(const KeyT& key, const ValueT& value)
{
    _double_list_link& b_head = table_[key.hash_value() % table_length_];
    _double_list_link* next = b_head.prev_ptr;
    iterator itor(this, 0);
    while(next)
    {
        HashEntity* entity = containerof_new(next, struct HashEntity, bucket_link_);
        if (entity->pair.key == key)
        {
            itor.entity_ = entity;  ////已经有这个key了
            return itor;

        }
        next = next->next_ptr;
    }
    HashEntity* new_entity = new HashEntity(key, value);
    dlh_right_push(&b_head, &new_entity->bucket_link_);
    dlh_right_push(&itor_root_, &new_entity->itor_link_);
    total_entity_num_++;
    itor.entity_ = new_entity;
    return itor;
}

template<typename KeyT, typename ValueT, unsigned int NumBuckets>
int HashMap<KeyT, ValueT, NumBuckets>::erase(const KeyT& key)
{
    _double_list_link& b_head = table_[key.hash_value() % table_length_];
    _double_list_link* next = b_head.prev_ptr;
    while(next)
    {
        HashEntity* entity = containerof_new(next, struct HashEntity, bucket_link_);
        if (entity->pair.key == key)
        {
            dlh_remove(&b_head, &entity->bucket_link_);
            dlh_remove(&itor_root_, &entity->itor_link_);
            total_entity_num_--;
            delete entity;
            return 0;  ///删除成功
        }
        next = next->next_ptr;
    }
    return -1; ////没有找到key
}
template<typename KeyT, typename ValueT, unsigned int NumBuckets>
int HashMap<KeyT, ValueT, NumBuckets>::erase(typename HashMap<KeyT, ValueT, NumBuckets>::iterator iter)
{
    HashEntity* entity = iter.entity_;
    iter.entity_ = 0;
    if (entity)
    {
        _double_list_link& b_head = table_[entity->pair.key.hash_value() % table_length_];
            
        dlh_remove(&b_head, &entity->bucket_link_);
        dlh_remove(&itor_root_, &entity->itor_link_);
        total_entity_num_--;
        delete entity;
        return 0;  ///删除成功
    }
    return -1; ////
}

template<typename KeyT, typename ValueT, unsigned int NumBuckets>
ValueT& HashMap<KeyT, ValueT, NumBuckets>::operator[] (const KeyT& key)
{
    _double_list_link& b_head = table_[key.hash_value() % table_length_];
    _double_list_link* next = b_head.prev_ptr;
    while(next)
    {
        HashEntity* entity = containerof_new(next, struct HashEntity, bucket_link_);
        if (entity->pair.key == key)
        {
            return entity->pair.value;
        }
        next = next->next_ptr;
    }
    ///没有找到，创建一个新节点
    HashEntity* new_entity = new HashEntity(key, ValueT());
    dlh_right_push(&b_head, &new_entity->bucket_link_);
    dlh_right_push(&itor_root_, &new_entity->itor_link_);
    total_entity_num_++;
    return new_entity->pair.value;
}

}
#endif