// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"strings"

	"go.etcd.io/raft/v3/raftpb"
)

func main() {
	cluster := flag.String("cluster", "http://127.0.0.1:9021", "comma separated cluster peers")
	id := flag.Int("id", 1, "node ID")
	// HTTP 服务器的 port 不是 raft kvstore 的 port
	kvport := flag.Int("port", 9121, "key-value server port")
	join := flag.Bool("join", false, "join an existing cluster")
	flag.Parse()

	// kvstore 写入数据
	// raftnode 读取数据
	//
	// proposeC 和 confChangeC 统一通过 ServeHTTP 方法获取提案，然后提交给 Raft 模块进行处理
	proposeC := make(chan string)
	defer close(proposeC)
	confChangeC := make(chan raftpb.ConfChange)
	defer close(confChangeC)

	// raft provides a commit stream for the proposals from the http api
	var kvs *kvstore
	// 将当前的 kvstore 序列化为 JSON 后返回
	getSnapshot := func() ([]byte, error) { return kvs.getSnapshot() }
	// 1. 创建 raft 节点
	// commitC 返回已经提交的日志条目
	// 这里为和 Raft 库的主要交互逻辑
	commitC, errorC, snapshotterReady := newRaftNode(*id, strings.Split(*cluster, ","), *join, getSnapshot, proposeC, confChangeC)

	// 2. 创建 KV 存储服务
	// 业务逻辑所在的地方，应该与 Raft 模块分离，或者说解耦
	//
	// 阻塞直到 snapshotterReady 准备好 snapshotter
	kvs = newKVStore(<-snapshotterReady, proposeC, commitC, errorC)

	// the key-value http handler will propose updates to raft
	// 3. 创建 HTTP 服务，提供 Restful API
	// 通过 HTTP 模块提供的 Restful API 向 Raft 模块提交 propose
	serveHTTPKVAPI(kvs, *kvport, confChangeC, errorC)
}
