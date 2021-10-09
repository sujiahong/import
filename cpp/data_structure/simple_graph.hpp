
#ifndef _SIMPLE_GRAPH_H_
#define _SIMPLE_GRAPH_H_

#include <unordered_set>

namespace su{

typedef struct EdgeNode
{
    struct EdgeNode* next_node;
    int data;
    unsigned int weight;

    EdgeNode()
    {
        next_node = NULL;
    }
    ~EdgeNode()
    {}
}EDGE_NODE_TYPE;

typedef struct Vertex
{
    struct EdgeNode* next_node;
    int data;   ///////顶点数据域  类型模板
    
    Vertex()
    {
        next_node = NULL;
    }
    ~Vertex()
    {}

    bool operator<(const struct Vertex& vex) const
    {
        return data < vex.data;
    }
}VERTEX_TYPE;

//template<typename T>
class Graph
{
    std::unordered_set<VERTEX_TYPE> m_vtx_set;
    unsigned int m_edge_num;
public:
    Graph();
    Graph(unsigned int a_node_num);
    ~Graph();

public:
    //初始化图
    int init();
    //清除图
    int clear();
    //插入顶点
    int insert_vertex(int a_data);
    int insert_vertex(const VERTEX_TYPE& a_vex);
    //插入边
    int insert_edge(int a_vex_data, int a_edge_data);
    int insert_edge(VERTEX_TYPE& a_vex, int a_edge_data);
    //删除顶点
    int remove_vertex(int a_data);
    int remove_vertex(const VERTEX_TYPE& a_vex);
    // 删除边
    int remove_edge(int a_vex_data, int a_edge_data);
    int remove_edge(const VERTEX_TYPE& a_vex, int a_edge_data);
    ////邻接顶点表
    int adjacent_vertex_list(std::vector<int>& a_vec);

    int get_vertex_edge_list(int a_vex_data, std::vector<int>& a_vec);

    unsigned int get_edge_num();

    bool is_adjacent(int a_vex_data_1, int a_vex_data_2);

private:

    
};

Graph::Graph():m_edge_num(0)
{}

Graph::Graph(unsigned int a_node_num):m_vtx_set(a_node_num),m_edge_num(0)
{}

Graph::~Graph()
{}

int Graph::init()
{
    clear();
}

int Graph::clear()
{
    for (auto itor = m_vtx_set.begin(); itor != m_vtx_set.end(); ++itor)
    {
        if (itor->next_node != NULL)
        {
            EDGE_NODE_TYPE* cur_ptr = itor->next_node;
            EDGE_NODE_TYPE* del_ptr = NULL;
            while (cur_ptr != NULL)
            {
                del_ptr = cur_ptr;
                cur_ptr = cur_ptr->next_node;
                delete del_ptr;
                del_ptr = NULL;
            }
        }
    }
    m_vtx_set.clear();
}

int Graph::insert_vertex(int a_data)
{
    VERTEX_TYPE vex;
    vex.data = a_data;
    m_vtx_set.insert(vex);
    return 0;
}
int Graph::insert_vertex(const VERTEX_TYPE& a_vex)
{
    m_vtx_set.insert(a_vex);
    return 0;
}

int Graph::insert_edge(int a_vex_data, int a_edge_data)
{
    VERTEX_TYPE vex;
    vex.data = a_vex_data;
    insert_edge(vex, a_edge_data);
    return 0;
}
int Graph::insert_edge(VERTEX_TYPE& a_vex, int a_edge_data)
{
    std::unordered_set<VERTEX_TYPE>::iterator itor = m_vtx_set.find(a_vex);
    if (itor != m_vtx_set.end())
    {
        EDGE_NODE_TYPE* cur_ptr = itor->next_node;
        while (cur_ptr != NULL)
        {
            if (a_edge_data == cur_ptr->data) break;
            if (cur_ptr->next_node == NULL)
            {
                EDGE_NODE_TYPE* node = new EdgeNode();
                node->data = a_edge_data;
                node->next_node = NULL;
                cur_ptr->next_node = node;
                break;
            }
            cur_ptr = cur_ptr->next_node;
        }
    }
    else
    {
        EDGE_NODE_TYPE* node = new EdgeNode();
        node->data = a_edge_data;
        node->next_node = NULL;
        a_vex.next_node = node;
        insert_vertex(a_vex);
    }
    return 0;
}

int Graph::remove_vertex(int a_data)
{
    VERTEX_TYPE vex;
    vex.data = a_data;
    remove_vertex(vex);
    return 0;
}
int Graph::remove_vertex(const VERTEX_TYPE& a_vex)
{
    std::unordered_set<VERTEX_TYPE>::iterator itor = m_vtx_set.find(a_vex);
    if (itor != m_vtx_set.end())
    {
        if (itor->next_node == NULL)
            m_vtx_set.erase(itor);
        else
        {
            EDGE_NODE_TYPE* cur_ptr = itor->next_node;
            EDGE_NODE_TYPE* del_ptr = NULL;
            while (cur_ptr != NULL)
            {
                del_ptr = cur_ptr;
                cur_ptr = cur_ptr->next_node;
                delete del_ptr;
                del_ptr = NULL;
            }
        }
    }
    for (auto it = m_vtx_set.begin(); it != m_vtx_set.end(); ++it)
    {
        if (a_vex.data != it->data && it->next_node != NULL)
        {
            remove_edge(*it, a_vex.data);
        }
    }
    return 0;
}

int Graph::remove_edge(int a_vex_data, int a_edge_data)
{
    VERTEX_TYPE vex;
    vex.data = a_vex_data;
    remove_edge(vex, a_edge_data);
    return 0;
}
int Graph::remove_edge(const VERTEX_TYPE& a_vex, int a_edge_data)
{
    std::unordered_set<VERTEX_TYPE>::iterator itor = m_vtx_set.find(a_vex);
    if (itor != m_vtx_set.end())
    {
        EDGE_NODE_TYPE* cur_ptr = itor->next_node;
        EDGE_NODE_TYPE* pre_ptr = cur_ptr;
        while (cur_ptr != NULL)
        {
            if (a_edge_data == cur_ptr->data)
            {
                pre_ptr->next_node = cur_ptr->next_node;
                delete cur_ptr;
            }
            pre_ptr = cur_ptr;
            cur_ptr = cur_ptr->next_node;
        }
    }
    return 0;
}

int Graph::adjacent_vertex_list(std::vector<int>& a_vec)
{
    for (auto it = m_vtx_set.begin(); it != m_vtx_set.end(); ++it)
    {
        a_vec.push_back(it->data);   
    }
}

int Graph::get_vertex_edge_list(int a_vex_data, std::vector<int>& a_vec)
{
    VERTEX_TYPE vex;
    vex.data = a_vex_data;
    std::unordered_set<VERTEX_TYPE>::iterator itor = m_vtx_set.find(vex);
    if (itor != m_vtx_set.end())
    {
        EDGE_NODE_TYPE* cur_ptr = itor->next_node;
        while (cur_ptr != NULL)
        {
            a_vec.push_back(cur_ptr->data);
            cur_ptr = cur_ptr->next_node;
        }        
    }
    return 0;
}
unsigned int Graph::get_edge_num()
{
    return m_edge_num;
}

bool Graph::is_adjacent(int a_vex_data_1, int a_vex_data_2)
{
    VERTEX_TYPE vex;
    vex.data = a_vex_data_1;
    std::unordered_set<VERTEX_TYPE>::iterator itor = m_vtx_set.find(vex);
    if (itor != m_vtx_set.end())
    {
        EDGE_NODE_TYPE* cur_ptr = itor->next_node;
        while (cur_ptr != NULL)
        {
            if (a_vex_data_2 == cur_ptr->data)
            {
                return true;
            }
            cur_ptr = cur_ptr->next_node;
        }        
    }
    return false;
}

}


#endif