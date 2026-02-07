# Project Structure

This document explains the organization of the Diting project.

## Directory Layout

```
diting/
├── README.md                   # Project overview (English)
├── README_CN.md                # Project overview (Chinese)
├── LICENSE                     # MIT License
├── CONTRIBUTING.md             # How to contribute
├── CHANGELOG.md                # Version history
├── CODE_OF_CONDUCT.md          # Community guidelines
│
├── python/                     # Python Implementation
│   ├── sentinel.py             # Main service
│   ├── sentinel_dns.py         # DNS hijacking module
│   ├── sentinel_ebpf.py        # eBPF monitoring module
│   ├── requirements.txt        # Python dependencies
│   └── README.md               # Python-specific documentation
│
├── cmd/                        # Go Applications
│   └── diting/                 # Main application
│       ├── main.go             # Entry point
│       ├── go.mod              # Go module definition
│       ├── go.sum              # Go dependencies checksum
│       └── README.md           # Go-specific documentation
│
├── pkg/                        # Go Packages (reusable libraries)
│   ├── dns/                    # DNS hijacking package
│   │   └── dnshijack.go
│   ├── waf/                    # WAF gateway package
│   │   └── wafgateway.go
│   └── ebpf/                   # eBPF monitoring package
│
├── deployments/                # Deployment Configurations
│   ├── docker/                 # Docker & Podman
│   │   ├── docker-compose.yml
│   │   ├── docker-compose-opensource.yml
│   │   ├── podman-compose.yml
│   │   ├── Dockerfile
│   │   ├── .dockerignore
│   │   └── .env.example
│   ├── coredns/                # CoreDNS configuration
│   │   └── Corefile
│   ├── nginx/                  # Nginx/OpenResty configuration
│   │   ├── nginx.conf
│   │   └── lua/
│   └── kubernetes/             # Kubernetes manifests (future)
│
├── docs/                       # Documentation
│   ├── STRUCTURE.md            # This file
│   ├── QUICKSTART.md           # Quick start guide
│   ├── INSTALL.md              # Installation guide
│   ├── ARCHITECTURE_DNS_HIJACK.md
│   ├── DEPLOYMENT_OPENSOURCE.md
│   ├── TECHNICAL_EBPF.md
│   ├── TEST.md
│   ├── DEMO.md
│   └── ...                     # Other technical docs
│
├── scripts/                    # Utility Scripts
│   ├── start-python.sh         # Start Python version
│   ├── start.sh                # Start Go version
│   ├── docker-deploy.sh        # Docker deployment
│   └── ...                     # Other utility scripts
│
├── logs/                       # Runtime Logs (gitignored)
│   └── audit.jsonl             # Audit trail
│
└── tests/                      # Test Files (future)
    ├── python/
    └── go/
```

## Design Principles

### 1. Language Separation
- **Python**: All Python code in `python/` directory
- **Go**: Following standard Go project layout (`cmd/`, `pkg/`)
- No mixing of languages in the same directory

### 2. Standard Go Layout
Following [golang-standards/project-layout](https://github.com/golang-standards/project-layout):
- `cmd/` - Main applications (entry points)
- `pkg/` - Library code that can be imported by external projects
- `internal/` - Private application code (future use)

### 3. Clear Separation of Concerns
- **Source code**: `python/`, `cmd/`, `pkg/`
- **Deployment**: `deployments/`
- **Documentation**: `docs/`
- **Utilities**: `scripts/`
- **Runtime data**: `logs/`

### 4. Documentation Organization
- **Root level**: Essential docs (README, LICENSE, CONTRIBUTING)
- **docs/**: Detailed technical documentation
- **Per-directory**: README.md for each major component

## Quick Start

### Python Version
```bash
cd python
pip install -r requirements.txt
python sentinel.py
```

### Go Version
```bash
cd cmd/diting
go run main.go
```

### Docker Deployment
```bash
cd deployments/docker
docker-compose up -d
```

## File Purposes

### Root Level Files

| File | Purpose |
|------|---------|
| `README.md` | Project overview (English, primary) |
| `README_CN.md` | Project overview (Chinese) |
| `LICENSE` | MIT License |
| `CONTRIBUTING.md` | Contribution guidelines |
| `CHANGELOG.md` | Version history |
| `CODE_OF_CONDUCT.md` | Community code of conduct |
| `.gitignore` | Git ignore rules |

### Python Directory

| File | Purpose |
|------|---------|
| `sentinel.py` | Main HTTP proxy service with AI analysis |
| `sentinel_dns.py` | DNS hijacking implementation |
| `sentinel_ebpf.py` | eBPF kernel-level monitoring |
| `requirements.txt` | Python package dependencies |
| `README.md` | Python-specific documentation |

### Go Directories

| Directory | Purpose |
|-----------|---------|
| `cmd/diting/` | Main application entry point |
| `pkg/dns/` | DNS hijacking library |
| `pkg/waf/` | WAF gateway library |
| `pkg/ebpf/` | eBPF monitoring library |

### Deployment Directory

| Directory | Purpose |
|-----------|---------|
| `deployments/docker/` | Docker and Podman configurations |
| `deployments/coredns/` | CoreDNS DNS server configuration |
| `deployments/nginx/` | Nginx/OpenResty reverse proxy config |
| `deployments/kubernetes/` | Kubernetes manifests (future) |

### Documentation Directory

| File | Purpose |
|------|---------|
| `QUICKSTART.md` | 5-minute quick start guide |
| `INSTALL.md` | Detailed installation instructions |
| `ARCHITECTURE_DNS_HIJACK.md` | DNS hijacking architecture |
| `DEPLOYMENT_OPENSOURCE.md` | Open-source deployment guide |
| `TECHNICAL_EBPF.md` | eBPF technical documentation |
| `TEST.md` | Testing guide |
| `DEMO.md` | Demo and presentation guide |

## Benefits of This Structure

1. **Professional**: Follows industry best practices
2. **Clear**: Easy to navigate and understand
3. **Scalable**: Easy to add new components
4. **Standard**: Familiar to developers from other projects
5. **Maintainable**: Clear separation makes maintenance easier

## Migration Notes

If you're looking for files that were previously in the root:
- Python files → `python/`
- Go files → `cmd/diting/` or `pkg/`
- Docker files → `deployments/docker/`
- Documentation → `docs/`
- Scripts → `scripts/`

## Contributing

When adding new code:
- Python code → `python/`
- Go applications → `cmd/`
- Go libraries → `pkg/`
- Deployment configs → `deployments/`
- Documentation → `docs/`
- Utility scripts → `scripts/`

See [CONTRIBUTING.md](../CONTRIBUTING.md) for detailed guidelines.
