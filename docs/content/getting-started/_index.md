---
title: "Getting Started"
weight: 1
bookFlatSection: false
bookCollapseSection: false
---

# Getting Started with Spoke

Welcome to Spoke! This section will help you get up and running quickly with the Spoke Protobuf Schema Registry.

## What You'll Learn

- [What is Spoke](/getting-started/what-is-spoke/) - Understanding Spoke and its purpose
- [Quick Start](/getting-started/quick-start/) - Get started in 5 minutes
- [Installation](/getting-started/installation/) - Complete installation guide
- [First Module](/getting-started/first-module/) - Create and manage your first protobuf module

## Prerequisites

Before you begin, ensure you have:

- Go 1.16 or later (for building from source)
- Protocol Buffers compiler (`protoc`)
- Language-specific protoc plugins (e.g., `protoc-gen-go` for Go)
- Basic understanding of Protocol Buffers

## Quick Installation

```bash
# Clone the repository
git clone https://github.com/platinummonkey/spoke.git
cd spoke

# Build the server and CLI
make build

# Start the server
./spoke -port 8080 -storage-dir ./storage
```

## Next Steps

Start with our [Quick Start Guide](/getting-started/quick-start/) to create your first protobuf module in Spoke.
