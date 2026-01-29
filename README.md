# 🚀 DistroKV

> A Distributed Key-Value Store built from scratch in Go.
> **Architecture:** Raft Consensus + LSM Tree + gRPC.

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![Architecture](https://img.shields.io/badge/Architecture-Raft%20%2B%20LSM-orange)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

## 📖 Overview

**DistroKV** is an educational distributed database designed to explore the internals of system consistency and storage engines. 

Unlike wrapping existing libraries (like `etcd` or `BadgerDB`), DistroKV implements the core algorithms **manually** to demonstrate deep understanding of distributed systems:

*   **The Brain (Consensus):** A custom implementation of the **Raft** consensus algorithm (Leader Election, Log Replication).
*   **The Memory (Storage):** A custom **LSM Tree** (Log-Structured Merge Tree) with MemTable, WAL (Write-Ahead Log), and SSTable flushing.
*   **The Nervous System:** **gRPC** with Protocol Buffers for typed node communication.

## 🏗️ Architecture

```mermaid
graph TD
    Client[Client CLI] -->|gRPC Put/Get| Server
    subgraph Server Node
        RPC[gRPC Handler]
        Raft[Raft Consensus Module]
        LSM[LSM Storage Engine]
        
        RPC -->|Submit Command| Raft
        Raft -->|Replicate| Peers[Other Nodes]
        Raft -->|Commit Details| LSM
        LSM -->|Write| WAL[WAL Log]
        LSM -->|Flush| SST[SSTable File]
    end
```

## 🚀 Getting Started

### Prerequisites
- Go 1.25+
- Protoc (Protocol Buffers Compiler)

### Installation
```bash
git clone https://github.com/ThaRealJozef/DistroKV.git
cd DistroKV
go mod tidy
```

### Running the Server
Start a single node (Leader automatically elected in single-node mode):
```powershell
go build -o bin/server.exe ./cmd/server
./bin/server.exe node1 50051
```

### Running the Client
Open a new terminal to interact with the store:
```powershell
go build -o bin/client.exe ./cmd/client

# Write Data
./bin/client.exe -addr localhost:50051 -op put -key user:101 -val "Jozef"

# Read Data
./bin/client.exe -addr localhost:50051 -op get -key user:101

### 🧪 Live Demo (Performance Check)
Run the verification script to demonstrate **memtable flushing, bloom filters, and compaction**:
```powershell
./verify_flush.ps1
```
```

## 🛠️ Technical Details

### Implemented Features
- **Raft Consensus:** 
    - Leader Election (Term management, randomized timeouts).
    - Log Replication (AppendEntries).
    - **Safety:** HardState persistence (`raft_state.json`).
- **LSM Storage Engine:** 
    - **MemTable:** Mutable in-memory map.
    - **WAL:** Crash recovery (auto-load on startup).
    - **SSTable:** Immutable disk files (Flushed >100 keys).
    - **Bloom Filters:** Fast lookups (skip disk if key missing).
    - **Compaction:** Background merge of SSTables (1-min interval).
- **Network:** gRPC + Protobuf (Binary Protocol).

### Limitations (Educational Scope)
- **Cluster Membership:** Static configuration (peers defined at startup).
- **Snapshotting:** Raft logs grow indefinitely (Log Compaction not implemented, but Storage Compaction is).

## 📝 License
MIT
