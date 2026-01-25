---
title: "API Reference"
weight: 7
bookFlatSection: false
bookCollapseSection: false
---

# API Reference

Complete API documentation for Spoke.

## Available Documentation

- [REST API Reference](/guides/api-reference/) - HTTP REST API endpoints
- [CLI Reference](/guides/cli-reference/) - Command-line interface
- [Authentication](/api/authentication/) - Auth methods and tokens
- [Webhooks](/api/webhooks/) - Webhook events and payloads
- [Error Codes](/api/errors/) - Error codes and handling

## API Overview

### Base URL

```
https://spoke.company.com
```

### Authentication

```http
Authorization: Bearer <token>
```

### Content Type

```http
Content-Type: application/json
```

### Rate Limits

| Plan | Requests/Hour |
|------|---------------|
| Free | 100 |
| Professional | 1,000 |
| Enterprise | 10,000 |

## Quick Examples

### List Modules

```bash
curl https://spoke.company.com/modules \
  -H "Authorization: Bearer $TOKEN"
```

### Push Module

```bash
curl -X POST https://spoke.company.com/modules/user/versions \
  -H "Authorization: Bearer $TOKEN" \
  -F "version=v1.0.0" \
  -F "file=@user.proto"
```

### Get Version

```bash
curl https://spoke.company.com/modules/user/versions/v1.0.0 \
  -H "Authorization: Bearer $TOKEN"
```

For complete API documentation, see the [REST API Reference](/guides/api-reference/).
