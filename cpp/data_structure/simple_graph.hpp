
#ifndef _SIMPLE_GRAPH_H_
#define _SIMPLE_GRAPH_H_

#include <unordered_set>

namespace su{

struct EdgeNode
{
    int data;
    unsigned int weight;
    struct EdgeNode* next_node;
};

typedef struct Vertex
{
    struct EdgeNode* next_node;
    int data;   ///////顶点数据域  类型模板
}VERTEX;

class Graph
{
    std::unordered_set<VERTEX> vtx_set;
public:
    Graph();
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

private:

};

Graph::Graph()
{}
Graph::~Graph()
{}

int Graph::init()
{

}

int Graph::clear()
{

}

}


#endif