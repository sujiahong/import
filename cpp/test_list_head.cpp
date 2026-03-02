#include <iostream>
#include "data_structure/list_head.h"

// 测试单链表
void test_single_list() {
    std::cout << "=== 测试单链表 ===" << std::endl;
    
    // 创建根节点
    _single_list_head root;
    root.next_ptr = nullptr;
    
    // 创建测试节点
    _single_list_head node1, node2, node3;
    node1.next_ptr = nullptr;
    node2.next_ptr = nullptr;
    node3.next_ptr = nullptr;
    
    // 测试 push 操作
    std::cout << "测试 push 操作" << std::endl;
    slh_push(&root, &node1);
    slh_push(&root, &node2);
    slh_push(&root, &node3);
    
    // 测试 pop 操作
    std::cout << "测试 pop 操作" << std::endl;
    _single_list_head* popped = slh_pop(&root);
    std::cout << "弹出节点: " << popped << std::endl;
    popped = slh_pop(&root);
    std::cout << "弹出节点: " << popped << std::endl;
    popped = slh_pop(&root);
    std::cout << "弹出节点: " << popped << std::endl;
    popped = slh_pop(&root); // 测试空链表情况
    std::cout << "弹出空链表: " << popped << std::endl;
    
    std::cout << "单链表测试完成" << std::endl;
    std::cout << std::endl;
}

// 测试双链表
void test_double_list() {
    std::cout << "=== 测试双链表 ===" << std::endl;
    
    // 创建根节点
    _double_list_head root;
    root.prev_ptr = nullptr;
    root.next_ptr = nullptr;
    
    // 测试 is_empty
    std::cout << "测试 is_empty: " << (dlh_is_empty(&root) ? "空" : "非空") << std::endl;
    
    // 创建测试节点
    _double_list_head node1, node2, node3, node4;
    node1.prev_ptr = nullptr;
    node1.next_ptr = nullptr;
    node2.prev_ptr = nullptr;
    node2.next_ptr = nullptr;
    node3.prev_ptr = nullptr;
    node3.next_ptr = nullptr;
    node4.prev_ptr = nullptr;
    node4.next_ptr = nullptr;
    
    // 测试 left_push 操作
    std::cout << "测试 left_push 操作" << std::endl;
    dlh_left_push(&root, &node1);
    dlh_left_push(&root, &node2);
    dlh_left_push(&root, &node3);
    std::cout << "左推后 is_empty: " << (dlh_is_empty(&root) ? "空" : "非空") << std::endl;
    
    // 测试 left_pop 操作
    std::cout << "测试 left_pop 操作" << std::endl;
    _double_list_head* popped = dlh_left_pop(&root);
    std::cout << "左弹节点: " << popped << std::endl;
    popped = dlh_left_pop(&root);
    std::cout << "左弹节点: " << popped << std::endl;
    popped = dlh_left_pop(&root);
    std::cout << "左弹节点: " << popped << std::endl;
    popped = dlh_left_pop(&root); // 测试空链表情况
    std::cout << "左弹空链表: " << popped << std::endl;
    std::cout << "左弹后 is_empty: " << (dlh_is_empty(&root) ? "空" : "非空") << std::endl;
    
    // 测试 right_push 操作
    std::cout << "测试 right_push 操作" << std::endl;
    dlh_right_push(&root, &node1);
    dlh_right_push(&root, &node2);
    dlh_right_push(&root, &node3);
    std::cout << "右推后 is_empty: " << (dlh_is_empty(&root) ? "空" : "非空") << std::endl;
    
    // 测试 right_pop 操作
    std::cout << "测试 right_pop 操作" << std::endl;
    popped = dlh_right_pop(&root);
    std::cout << "右弹节点: " << popped << std::endl;
    popped = dlh_right_pop(&root);
    std::cout << "右弹节点: " << popped << std::endl;
    popped = dlh_right_pop(&root);
    std::cout << "右弹节点: " << popped << std::endl;
    popped = dlh_right_pop(&root); // 测试空链表情况
    std::cout << "右弹空链表: " << popped << std::endl;
    std::cout << "右弹后 is_empty: " << (dlh_is_empty(&root) ? "空" : "非空") << std::endl;
    
    // 测试插入操作
    std::cout << "测试插入操作" << std::endl;
    dlh_right_push(&root, &node1);
    dlh_right_push(&root, &node3);
    dlh_left_insert(&root, &node3, &node2); // 在 node3 左侧插入 node2
    dlh_right_insert(&root, &node1, &node4); // 在 node1 右侧插入 node4
    
    // 测试 remove 操作
    std::cout << "测试 remove 操作" << std::endl;
    dlh_remove(&root, &node2);
    dlh_remove(&root, &node4);
    
    // 测试 is_correlation
    std::cout << "测试 is_correlation" << std::endl;
    std::cout << "node1 与链表相关: " << (dlh_is_correlation(&root, &node1) ? "是" : "否") << std::endl;
    std::cout << "node2 与链表相关: " << (dlh_is_correlation(&root, &node2) ? "是" : "否") << std::endl;
    std::cout << "node3 与链表相关: " << (dlh_is_correlation(&root, &node3) ? "是" : "否") << std::endl;
    std::cout << "node4 与链表相关: " << (dlh_is_correlation(&root, &node4) ? "是" : "否") << std::endl;
    
    // 测试边界情况 - 空指针
    std::cout << "测试边界情况 - 空指针" << std::endl;
    // 注意：这些测试可能会导致崩溃，因为原代码没有空指针检查
    // slh_push(nullptr, &node1); // 可能崩溃
    // slh_pop(nullptr); // 可能崩溃
    // dlh_left_push(nullptr, &node1); // 可能崩溃
    // dlh_right_push(nullptr, &node1); // 可能崩溃
    // dlh_left_pop(nullptr); // 可能崩溃
    // dlh_right_pop(nullptr); // 可能崩溃
    // dlh_is_empty(nullptr); // 可能崩溃
    // dlh_is_correlation(nullptr, &node1); // 可能崩溃
    // dlh_remove(nullptr, &node1); // 可能崩溃
    
    std::cout << "双链表测试完成" << std::endl;
    std::cout << std::endl;
}

int main() {
    test_single_list();
    test_double_list();
    std::cout << "所有测试完成" << std::endl;
    return 0;
}
