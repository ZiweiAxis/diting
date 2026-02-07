# Python Implementation

This directory contains the Python implementation of Diting.

## Quick Start

```bash
# Install dependencies
pip install -r requirements.txt

# Start the service
python sentinel.py
```

## Files

- `sentinel.py` - Main service (HTTP proxy + AI analysis)
- `sentinel_dns.py` - DNS hijacking module
- `sentinel_ebpf.py` - eBPF kernel monitoring module
- `requirements.txt` - Python dependencies

## Features

- ✅ Zero-intrusion deployment
- ✅ AI-powered intent analysis
- ✅ Risk assessment engine
- ✅ Human approval workflow
- ✅ Full audit trail

## Configuration

Edit `sentinel.py` to configure:
- OpenAI API key
- Risk thresholds
- Approval workflow
- Logging settings

## Testing

```bash
# Safe request
curl http://localhost:8080/get

# Dangerous request (requires approval)
curl -X DELETE http://localhost:8080/delete
```

## Performance

- Throughput: ~200 req/s
- Latency: < 20ms (low risk), < 2s (high risk)
- Memory: ~50 MB

## Production Deployment

For production use, consider:
1. Using the Go implementation (higher performance)
2. Running behind a reverse proxy (Nginx)
3. Using a proper database for audit logs
4. Implementing high availability

## See Also

- [Go Implementation](../cmd/diting/)
- [Docker Deployment](../deployments/docker/)
- [Documentation](../docs/)
