/*
链表实现不进行判空，使用者保证指针正确
*/

#ifndef __LIST_HEAD_H__
#define __LIST_HEAD_H__

struct _single_list_head
{
    _single_list_head* next_ptr;
};

inline void slh_push(_single_list_head* a_root, _single_list_head* a_node)
{
    _single_list_head* tmp_ptr = a_root->next_ptr;
    a_root->next_ptr = a_node;
    a_node->next_ptr = tmp_ptr;
}

inline _single_list_head* slh_pop(_single_list_head* a_root)
{
     _single_list_head* tmp_ptr = a_root->next_ptr;
     if (a_root->next_ptr) a_root->next_ptr = a_root->next_ptr->next_ptr;
     return tmp_ptr;
}

struct _double_list_head
{
    _double_list_head* prev_ptr;
    _double_list_head* next_ptr;
};

#define INIT_HEAD(ptr) \
    ptr->prev_ptr = 0; \
    ptr->next_ptr = 0;

inline bool dlh_is_empty(struct _double_list_head *a_root)
{
	return (0 == a_root->prev_ptr && 0 == a_root->next_ptr);
};

inline void dlh_left_push(_double_list_head* a_root, _double_list_head* a_node)
{
    if (a_root->prev_ptr)
    {
        a_root->prev_ptr->prev_ptr = a_node;
        a_node->next_ptr = a_root->prev_ptr;
        a_root->prev_ptr = a_node;
        a_node->prev_ptr = 0;
    }
    else
    {
        a_root->prev_ptr = a_node;
        a_root->next_ptr = a_node;
        INIT_HEAD(a_node)
    }
}

inline _double_list_head* dlh_left_pop(_double_list_head* a_root)
{
    _double_list_head* tmp_ptr = a_root->prev_ptr;
    if (tmp_ptr)
    {
        if (tmp_ptr->next_ptr)
        {
            a_root->prev_ptr = tmp_ptr->next_ptr;
            a_root->prev_ptr->prev_ptr = 0;
            tmp_ptr->next_ptr = 0;
        }
        else
            INIT_HEAD(a_root)
    }
    return tmp_ptr;
}

inline void dlh_left_insert(_double_list_head* a_root, _double_list_head* a_right_node, _double_list_head* a_node)
{
    if (a_right_node->prev_ptr)
        a_right_node->prev_ptr->next_ptr = a_node;
    else
        a_root->prev_ptr = a_node;
    a_node->prev_ptr = a_right_node->prev_ptr;
    a_node->next_ptr = a_right_node;
    a_right_node->prev_ptr = a_node;
}

inline void dlh_right_push(_double_list_head* a_root, _double_list_head* a_node)
{
    if (a_root->next_ptr)
    {
        a_root->next_ptr->next_ptr = a_node;
        a_node->prev_ptr = a_root->next_ptr;
        a_root->next_ptr = a_node;
        a_node->next_ptr = 0;
    }
    else
    {
        a_root->prev_ptr = a_node;
        a_root->next_ptr = a_root;
        INIT_HEAD(a_node)
    }
}

inline _double_list_head* dlh_right_pop(_double_list_head* a_root)
{
    _double_list_head* tmp_ptr = a_root->next_ptr;
    if (tmp_ptr)
    {
        if (tmp_ptr->prev_ptr)
        {
            a_root->next_ptr = tmp_ptr->prev_ptr;
            a_root->next_ptr->next_ptr = 0;
            tmp_ptr->prev_ptr = 0;
        }
        else
            INIT_HEAD(a_root)
    }
    return tmp_ptr;
}

inline void dlh_right_insert(_double_list_head* a_root, _double_list_head* a_left_node, _double_list_head* a_node)
{
    if (a_left_node->next_ptr)
        a_left_node->next_ptr->prev_ptr = a_node;
    else
        a_root->next_ptr = a_node;
    a_node->prev_ptr = a_left_node;
    a_node->next_ptr = a_left_node->next_ptr;
    a_left_node->next_ptr = a_node;
}

inline void dlh_remove(_double_list_head* a_root, _double_list_head* a_node)
{
    if (a_node->prev_ptr)
        a_node->prev_ptr->next_ptr = a_node->next_ptr;
    else
        a_root->prev_ptr = a_node->next_ptr;
    if (a_node->next_ptr)
        a_node->next_ptr->prev_ptr = a_node->prev_ptr;
    else
        a_root->next_ptr = a_node->prev_ptr;
    INIT_HEAD(a_node)
}

inline bool dlh_is_empty(_double_list_head* a_root)
{
    return (0 == a_root->prev_ptr && 0 == a_root->next_ptr);
}

inline bool dlh_is_correlation(_double_list_head* a_root, _double_list_head* a_node)
{
    return (a_node->prev_ptr || a_node->next_ptr || (a_node == a_root->prev_ptr) || (a_node == a_root->next_ptr));
}



#endif