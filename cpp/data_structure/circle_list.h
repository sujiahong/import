#ifndef _CIRCLE_LIST_HPP_
#define _CIRCLE_LIST_HPP_

namespace su
{
template<typename T>
class CircleQueue
{
public:
    typedef T* _ptr_type;
private:
    _ptr_type m_elements_;
    unsigned int m_size_;
    unsigned int m_head_;
    unsigned int m_tail_;
public:
    CircleQueue():m_elements_(NULL),m_size_(0),m_head_(0),m_tail_(0)
    {}

    ~CircleQueue()
    {
        Destory();
    }

    bool Create(unsigned int a_size)
    {
        if (a_size != (unsigned int)-1)
        {
            m_size_ = a_size + 1;
        }
        else
        {
            m_size_ = a_size;
        }
        m_elements_ = new T[m_size_];
        return (m_elements_ == NULL) ? false : true;
    }

    void Destory()
    {
        if (m_elements_)
            delete[] m_elements_;
        m_elements_ = NULL;
        m_size_ = 0;
        m_head_ = 0;
        m_tail_ = 0;
    }
    void Clear()
    {
        m_head_ = 0;
        m_tail_ = 0;
    }
    ///尾进
    bool Push(T& a_ele)
    {
        unsigned int pos = (m_tail_+1)%m_size_;
        if (pos == m_head_) return false;
        m_elements_[m_tail_] = a_ele;
        m_tail_ = pos;
        return true;
    } 
    //头出
    bool Pop(T& a_ele)
    {
        if (m_head_ == m_tail_) return false;
        a_ele = m_elements_[m_head_];
        m_head_ = (m_head_+1)%m_size_;
        return true;
    }
    //取头部值，不弹出
    bool PeekFront(T& a_ele)
    {
        if (m_head_ == m_tail_) return false;
        a_ele = m_elements_[m_head_];
        return true;
    }
    //只是删除值，不取值
    bool RemoveFront()
    {
        if (m_head_ == m_tail_) return false;
        m_head_ = (m_head_+1)%m_size_;
        return true;
    }
    unsigned int GetFreeSpace()
    {
        if (m_head_ > m_tail_)
            return m_head_ - m_tail_;
        return m_size_ - (m_tail_ - m_head_) -1;
    }
    //批量尾进
    bool Push(const T* a_ele_ptr, unsigned int a_size)
    {
        if (GetFreeSpace() < a_size ||  a_size == 0) return false;
        unsigned int pos = (m_tail_+a_size)%m_size_;
        if (pos == m_head_) return false;
        if (pos > m_tail_)
        {
            if (a_ele_ptr)
                memcpy((void*)&m_elements_[m_tail_], (void*)a_ele_ptr, (unsigned long int)a_size*sizeof(T));
        }
        else
        {
            if (a_ele_ptr)
            {
                unsigned long int sz = m_size_ - m_tail_;
                memcpy((void*)&m_elements_[m_tail_], (void*)a_ele_ptr, (unsigned long int)sz*sizeof(T));
                unsigned long int rs = a_size - sz;
                memcpy((void*)m_elements_, (void*)&a_ele_ptr[sz], (unsigned long int)rs*sizeof(T));
            }
        }
        m_tail_ = pos;
        return true;
    }
    //批量头出
    bool Pop(T* a_ele_ptr, unsigned int a_size)
    {
        if (m_head_ == m_tail_ || a_size == 0) return false;
        if (GetQueueLen() < a_size) return false;
        unsigned int pos = (m_head_+a_size) % m_size_;
        if (pos > m_head_)
        {
            if (a_ele_ptr)
                memcpy((void*)a_ele_ptr, (void*)&(m_elements_[m_head_]), (unsigned long int)a_size*sizeof(T));
        }
        else
        {
            if (a_ele_ptr)
            {
                unsigned long int sz = m_size_ - m_head_;
                memcpy((void*)a_ele_ptr, (void*)&(m_elements_[m_head_]), (unsigned long int)sz*sizeof(T));
                unsigned long int rs = pos - sz;
                memcpy((void*)&a_ele_ptr[sz], (void*)m_elements_, (unsigned long int)rs*sizeof(T));
            }
        }
        m_head_ = pos;
        return true;
    }
    //批量取偏移过的数据
    bool GetData(T* a_ele_ptr, unsigned int a_size, unsigned int a_offset=0)
    {
        if (m_head_ == m_tail_ || a_size == 0) return false;
        if (GetQueueLen() < a_size+a_offset) return false;
        unsigned int start_pos = (m_head_ + a_offset) % m_size_;
        unsigned int pos = (start_pos + a_size) % m_size_;
        if (pos > start_pos)
        {
            if (a_ele_ptr)
                memcpy((void*)a_ele_ptr, (void*)&(m_elements_[start_pos]), (unsigned long int)a_size*sizeof(T));
        }
        else
        {
            if (a_ele_ptr)
            {
                unsigned long int sz = m_size_ - start_pos;
                memcpy((void*)a_ele_ptr, (void*)&(m_elements_[start_pos]), (unsigned long int)sz*sizeof(T));
                unsigned long int rs = pos - sz;
                memcpy((void*)&a_ele_ptr[sz], (void*)m_elements_, (unsigned long int)rs*sizeof(T));
            }
        }
        return true;
    }

    bool GetQueueLen()
    {
        if (m_head_ > m_tail_)
        {
            return m_size_ - (m_head_ - m_tail_);
        }
        return m_tail_ - m_head_;
    }

    bool GetMaxQueueLen()
    {
        return m_size_-1;
    }

    bool IsFull()
    {
		unsigned int next_pos = (m_tail_ + 1) % m_size_;
		if(next_pos == m_head_)
			return true;
		return false;
    }

    bool IsEmpty()
    {
		if(m_head_ == m_tail_)
			return true;
		return false;
    }

};

}


#endif