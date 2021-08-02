
#ifndef _SIMPLE_GRAPH_H_
#define _SIMPLE_GRAPH_H_

#include <unordered_set>

namespace su{

struct EdgeNode
{
    struct EdgeNode* next_node;
    int data;
    unsigned int weight;
};

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
}VERTEX;

//template<typename T>
class Graph
{
    std::unordered_set<VERTEX> m_vtx_set;
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
    int insert_vertex(int data);
    //插入边
    int insert_edge(int from_data, int to_data);
    //删除顶点
    int remove_vertex(int data);
    // 删除边
    int remove_edge(int from_data, int to_data);
    ////邻接顶点表
    int adjacent_vertex_list();

    int get_vertex_edge_list(int data, std::vector<int> a_vec);

    unsigned int get_vertex_edge_num();

private:

    
};

Graph::Graph()
{}

Graph::Graph(unsigned int a_node_num):m_vtx_set(a_node_num)
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
            struct EdgeNode* cur_ptr = itor->next_node;
            struct EdgeNode* del_ptr = NULL;
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
    VERTEX node;
    node.data = a_data;
    m_vtx_set.insert(node);
}

int Graph::insert_edge(int from_data, int to_data)
{

}

int Graph::remove_vertex(int data)
{

}

int Graph::remove_edge(int from_data, int to_data)
{

}

int Graph::adjacent_vertex_list()
{

}

}


#endif