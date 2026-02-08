# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

## Reporting a Vulnerability

We take security issues seriously. If you believe you have found a security vulnerability, please report it responsibly.

- **Preferred**: Open a [GitHub Security Advisory](https://github.com/hulk-yin/diting/security/advisories/new) (private by default).
- **Alternative**: Open a public [Issue](https://github.com/hulk-yin/diting/issues) with a clear description; avoid posting exploit details until a fix is available.

Please include:

- Description of the vulnerability and impact
- Steps to reproduce (if possible)
- Suggested fix or mitigation (optional)

We will acknowledge reports and work on a fix; we may ask for more detail. We do not support disclosure of vulnerabilities only via private channels outside GitHub unless previously agreed.

## Configuration Security

- Do **not** commit `config.json` or any file containing `app_secret`, `api_key`, or other credentials. Use `config.example.json` as a template and keep real config local or in environment variables.
- Rotate any credentials that may have been exposed in repository history.
