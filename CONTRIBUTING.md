# Contributing to Sentinel-AI

Thank you for your interest in contributing to Sentinel-AI! 🎉

## How to Contribute

### Reporting Bugs

If you find a bug, please open an issue with:
- A clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Your environment (OS, Python/Go version, etc.)

### Suggesting Features

We welcome feature suggestions! Please open an issue with:
- A clear description of the feature
- Use cases and benefits
- Any implementation ideas you have

### Submitting Pull Requests

1. **Fork the repository**
   ```bash
   git clone https://github.com/hulk-yin/sentinel-ai.git
   cd sentinel-ai
   ```

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Follow the existing code style
   - Add tests if applicable
   - Update documentation

4. **Test your changes**
   ```bash
   # Python version
   python sentinel.py
   
   # Go version
   go run main.go
   ```

5. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

6. **Push and create a PR**
   ```bash
   git push origin feature/your-feature-name
   ```

## Code Style

### Python
- Follow PEP 8
- Use type hints where possible
- Add docstrings for functions and classes

### Go
- Follow Go conventions
- Run `go fmt` before committing
- Add comments for exported functions

## Commit Message Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

## Development Setup

### Prerequisites
- Python 3.8+ or Go 1.21+
- Docker (for testing)
- Git

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/hulk-yin/sentinel-ai.git
   cd sentinel-ai
   ```

2. **Install dependencies**
   ```bash
   # Python
   pip install -r requirements.txt
   
   # Go
   go mod download
   ```

3. **Run tests**
   ```bash
   # Python
   python -m pytest
   
   # Go
   go test ./...
   ```

## Questions?

Feel free to open an issue or reach out to the maintainers!

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
