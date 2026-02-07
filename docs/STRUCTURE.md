# Diting Project Structure

```
diting/
├── README.md                   # English documentation (primary)
├── README_CN.md                # Chinese documentation
├── LICENSE                     # MIT License
├── CONTRIBUTING.md             # Contribution guidelines
├── CHANGELOG.md                # Version history
├── CODE_OF_CONDUCT.md          # Community guidelines
│
├── python/                     # Python implementation
│   ├── sentinel.py             # Main Python service
│   ├── sentinel_dns.py         # DNS hijacking module
│   ├── sentinel_ebpf.py        # eBPF monitoring module
│   ├── requirements.txt        # Python dependencies
│   └── README.md               # Python-specific docs
│
├── go/                         # Go implementation (future)
│   └── README.md               # Go-specific docs
│
├── cmd/                        # Command-line applications
│   └── diting/                 # Main Go application
│       └── main.go             # Entry point
│
├── pkg/                        # Go packages (libraries)
│   ├── dns/                    # DNS hijacking package
│   │   └── dnshijack.go
│   ├── waf/                    # WAF gateway package
│   │   └── wafgateway.go
│   └── ebpf/                   # eBPF monitoring package
│       └── README.md
│
├── deployments/                # Deployment configurations
│   ├── docker/                 # Docker deployment
│   │   ├── docker-compose.yml
│   │   ├── docker-compose-opensource.yml
│   │   └── podman-compose.yml
│   ├── kubernetes/             # Kubernetes manifests (future)
│   │   └── README.md
│   ├── coredns/                # CoreDNS configuration
│   │   └── Corefile
│   └── nginx/                  # Nginx/OpenResty configuration
│       └── lua/
│
├── docs/                       # Documentation
│   ├── QUICKSTART.md
│   ├── INSTALL.md
│   ├── ARCHITECTURE_DNS_HIJACK.md
│   ├── TECHNICAL_EBPF.md
│   ├── DEPLOYMENT_OPENSOURCE.md
│   ├── TEST.md
│   └── DEMO.md
│
├── scripts/                    # Utility scripts
│   ├── start-python.sh
│   ├── start-go.sh
│   └── test.sh
│
├── logs/                       # Audit logs (gitignored)
│   └── audit.jsonl
│
└── tests/                      # Test files
    ├── python/
    └── go/
```

## Architecture Principles

### 1. Language Separation
- **Python**: `python/` directory - Quick prototyping, easy deployment
- **Go**: `cmd/` and `pkg/` - High performance, production-ready

### 2. Standard Go Layout
Following [golang-standards/project-layout](https://github.com/golang-standards/project-layout):
- `cmd/` - Main applications
- `pkg/` - Library code (can be imported by external projects)
- `internal/` - Private application code (future)

### 3. Deployment Separation
- `deployments/` - All deployment configs in one place
- Separate subdirectories for Docker, Kubernetes, etc.

### 4. Documentation Organization
- Root: Essential docs (README, LICENSE, CONTRIBUTING)
- `docs/`: Detailed technical documentation

## Migration Guide

### For Python Users
```bash
cd python
pip install -r requirements.txt
python sentinel.py
```

### For Go Users
```bash
cd cmd/diting
go run main.go
```

### For Docker Users
```bash
cd deployments/docker
docker-compose up -d
```

## Benefits

1. **Clear Separation**: Python and Go code don't mix
2. **Standard Layout**: Follows Go community best practices
3. **Easy Navigation**: Developers can quickly find what they need
4. **Scalable**: Easy to add new languages or components
5. **Professional**: Looks like a mature open-source project
