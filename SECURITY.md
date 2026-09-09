# Security Policy

The Rune team takes the security of our software and users seriously. This document outlines our policy for reporting security vulnerabilities.

---

## Supported Versions

Security updates and patches are actively provided for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| `0.1.x` | :white_check_mark: |
| `< 0.1` | :x:                |

We strongly recommend always running the latest released version of Rune.

---

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you discover a security vulnerability in Rune, please report it confidentially via one of the following methods:

1. **GitHub Private Vulnerability Reporting** *(Preferred)*:
   - Navigate to the [Security Advisories](https://github.com/octopyid/rune/security/advisories/new) page.
   - Click **"Report a vulnerability"** to submit a private disclosure.

2. **Email**:
   - Send details to **security@octopy.dev**.
   - Use the subject line: `[SECURITY] Vulnerability report for Rune`.

### What to Include in Your Report

To help us triage and resolve the issue quickly, please provide:
- A clear description of the vulnerability.
- Steps to reproduce the issue (including sample `Runefile` configurations or shell commands, if applicable).
- An explanation of the potential impact and attack vectors.
- Any suggested fixes or mitigations (optional).

---

## Response & Disclosure Process

1. **Acknowledgement**: We will acknowledge receipt of your report within **48 hours**.
2. **Assessment & Confirmation**: We will investigate and confirm whether the issue is a valid security vulnerability.
3. **Fix & Verification**: We will develop and test a fix in a private branch.
4. **Release & Advisory**: We will publish a patched release along with a public GitHub Security Advisory crediting your discovery (unless you prefer to remain anonymous).

Thank you for helping keep Rune and its community secure!
