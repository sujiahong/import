//
//  common_utility.h
//  sea_route_division_rpc
//
//  Created by 周斌 on 2020/4/30.
//  Copyright © 2020 周斌. All rights reserved.
//

#ifndef UTIL_COMMON_UTILITY_H_
#define UTIL_COMMON_UTILITY_H_

// #include <google/protobuf/message.h>            // ::google::protobuf::Message
// #include <google/protobuf/repeated_field.h>     // ::google::protobuf::RepeatedField

#include <string>
#include <vector>
#include <list>
#include <map>
#include <set>
#include <sstream>

// #include "protodef/MessageType.pb.h"

namespace su {

template<class T>
std::string ObjToString(const T& obj);

template<class T>
std::string ObjVecToString(const std::vector<T>& obj_vec,
                           size_t max_disp_count = 5) {
    std::ostringstream oss;
    oss << "[";
    for (int32_t i = 0; i < obj_vec.size(); i++) {
        if (i >= max_disp_count) {
            break;
        }
        if (i > 0) {
            oss << ", ";
        }
        oss << obj_vec[i];
    }
    if (obj_vec.size() > max_disp_count) {
        oss << ", (" << obj_vec.size() - max_disp_count << " more)...";
    }
    oss << "]";
    return oss.str();
}

template<class T>
std::string ObjListToString(const std::list<T>& obj_list,
                            size_t max_disp_count = 5) {
    std::ostringstream oss;
    oss << "[";
    int32_t i = 0;
    typename std::list<T>::const_iterator it = obj_list.begin();
    while (it != obj_list.end()) {
        if (i >= max_disp_count) {
            break;
        }
        if (i > 0) {
            oss << ", ";
        }
        oss << *it;
        i++;
        it++;
    }
    if (obj_list.size() > max_disp_count) {
        oss << ", (" << obj_list.size() - max_disp_count << " more)...";
    }
    oss << "]";
    return oss.str();
}

template<class K, class V>
std::string ObjMapToString(const std::map<K, V>& obj_msp,
                           size_t max_disp_count = 5) {
    std::ostringstream oss;
    oss << "{";
    int32_t i = 0;
    typename std::map<K, V>::const_iterator it = obj_msp.begin();
    while (it != obj_msp.end()) {
        if (i >= max_disp_count) {
            break;
        }
        if (i > 0) {
            oss << ", ";
        }
        oss << it->first << ":" << it->second;
        i++;
        it++;
    }
    if (obj_msp.size() > max_disp_count) {
        oss << ", (" << obj_msp.size() - max_disp_count << " more)...";
    }
    oss << "}";
    return oss.str();
}

template<class T>
std::string ObjSetToString(const std::set<T>& obj_set, int32_t max_disp_count = 5) {
    std::ostringstream oss;
    oss << "{";
    int32_t i = 0;
    typename std::set<T>::const_iterator it = obj_set.begin();
    while (it != obj_set.end()) {
        if (max_disp_count > 0 && i >= max_disp_count) {
            break;
        }
        if (i > 0) {
            oss << ", ";
        }
        oss << *it;
        i++;
        it++;
    }
    if (max_disp_count > 0 && obj_set.size() > max_disp_count) {
        oss << ", (" << obj_set.size() - max_disp_count << " more)...";
    }
    oss << "}";
    return oss.str();
}


template<class T>
std::string ObjVecToString(const ::google::protobuf::RepeatedField<T>& obj_vec,
                           int32_t max_disp_count = 5) {
    std::ostringstream oss;
    oss << "[";
    for (int32_t i = 0; i < obj_vec.size(); i++) {
        if (i >= max_disp_count) {
            break;
        }
        if (i > 0) {
            oss << ", ";
        }
        oss << obj_vec.Get(i);
    }
    if (obj_vec.size() > max_disp_count) {
        oss << ", (" << obj_vec.size() - max_disp_count << " more)...";
    }
    oss << "]";
    return oss.str();
}

inline std::ostream& operator<<(std::ostream& os,
                                const ::google::protobuf::Message& msg) {
    os << msg.ShortDebugString();
    return os;
}

bool IsDecimal(const std::string& str);

inline void SetErrInfo(MessageType::ErrorInfo* error,
                       const int64_t error_code,
                       const std::string& error_msg = "",
                       const int32_t line_number = 0) {
    if (error != NULL) {
        if (error_msg.length() > 0) {
            error->set_err_msg(error_msg);
        }
        error->set_err_code(error_code);
    }
}

inline void CopyErrInfo(MessageType::ErrorInfo* dst,
                        const MessageType::ErrorInfo* src) {
    if (dst != NULL) {
        if (src != NULL) {
            dst->CopyFrom(*src);
        } else {
            dst->Clear();
        }
    }
}

inline void ClearErrInfo(MessageType::ErrorInfo* error) {
    if (error != NULL) {
        error->Clear();
    }
}

}   // namespace common_utility

#endif  // UTIL_COMMON_UTILITY_H_
