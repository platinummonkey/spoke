---
title: "Architecture"
weight: 5
bookFlatSection: false
bookCollapseSection: false
---

# Architecture

Technical architecture and design of Spoke.

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Spoke Registry                          │
│                                                                 │
│  ┌───────────────┐  ┌──────────────┐  ┌────────────────────┐  │
│  │   HTTP API    │  │   Web UI     │  │  Webhook System    │  │
│  │   (REST)      │  │   (React)    │  │                    │  │
│  └───────┬───────┘  └──────┬───────┘  └─────────┬──────────┘  │
│          │                  │                     │             │
│  ┌───────┴──────────────────┴─────────────────────┴──────────┐ │
│  │              Core Business Logic Layer                     │ │
│  │  ┌──────────┐ ┌────────┐ ┌──────┐ ┌──────┐ ┌─────────┐  │ │
│  │  │ Modules  │ │Versions│ │ Auth │ │ RBAC │ │  Audit  │  │ │
│  │  └──────────┘ └────────┘ └──────┘ └──────┘ └─────────┘  │ │
│  └────────────────────────────────────────────────────────────┘ │
│          │                  │                     │             │
│  ┌───────┴──────────┐  ┌────┴────────┐  ┌────────┴──────────┐ │
│  │ Storage Layer    │  │  Database   │  │  Cache (Redis)    │ │
│  │ (Filesystem)     │  │ (PostgreSQL)│  │                   │ │
│  └──────────────────┘  └─────────────┘  └───────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    │                   │
              ┌─────▼──────┐      ┌────▼─────┐
              │  Sprocket  │      │  Clients │
              │ (Compiler) │      │   (CLI)  │
              └────────────┘      └──────────┘
```

## Components

### HTTP API Server
- RESTful API for module and version management
- Authentication and authorization
- Webhook dispatching
- Health checks and metrics

### Storage Layer
- Filesystem-based module storage
- Organized by module → version hierarchy
- Stores proto files and compiled artifacts

### Database (PostgreSQL)
- User accounts and authentication
- Organizations and multi-tenancy
- RBAC roles and permissions
- Audit logs
- Metadata storage

### Cache (Redis)
- Permission caching
- Session storage
- API response caching
- Reduces database load

### Sprocket Service
- Background compilation worker
- Monitors for new proto files
- Compiles to multiple languages
- Dependency resolution

### Web UI
- React-based interface
- Module browsing
- Version history
- Schema visualization

### CLI Tool
- Command-line interface
- Push/pull operations
- Local compilation
- Validation

## Documentation Sections

- [System Architecture](/architecture/system/) - Overall system design
- [Storage Layer](/architecture/storage/) - File storage design
- [API Design](/architecture/api/) - REST API architecture
- [Compilation Service](/architecture/sprocket/) - Sprocket worker design
- [Security](/architecture/security/) - Security model
- [Performance](/architecture/performance/) - Performance considerations
