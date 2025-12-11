# API Contracts: Fix WASM Plugin Delay Fault Bug

**Feature**: 010-fix-wasm-delay-bug  
**Date**: 2025-12-11

## Overview

此功能的 API 契约变更仅涉及 Policy 结构中的 `DelayAction` 字段类型变更。

## Policy Schema Changes

### DelayAction Schema

**Before**:
```json
{
  "delay": {
    "fixed_delay": "string"
  }
}
```

**After**:
```json
{
  "delay": {
    "fixed_delay_ms": "integer"
  }
}
```

## OpenAPI Fragment

```yaml
components:
  schemas:
    DelayAction:
      type: object
      properties:
        fixed_delay_ms:
          type: integer
          format: int64
          minimum: 0
          maximum: 30000
          description: Delay duration in milliseconds
          example: 500
      required:
        - fixed_delay_ms

    Fault:
      type: object
      properties:
        percentage:
          type: integer
          minimum: 0
          maximum: 100
          description: Probability of fault injection (0-100%)
        abort:
          $ref: '#/components/schemas/AbortAction'
        delay:
          $ref: '#/components/schemas/DelayAction'
        start_delay_ms:
          type: integer
          description: Wait time before injecting fault (ms)
        duration_seconds:
          type: integer
          description: Policy expiration time (0 = permanent)
```

## Example Request

### Create Policy with Delay Fault

```http
POST /v1/policies HTTP/1.1
Host: hfi-control-plane:8080
Content-Type: application/json

{
  "metadata": {
    "name": "delay-demo"
  },
  "spec": {
    "selector": {
      "service": "frontend",
      "namespace": "demo"
    },
    "rules": [
      {
        "match": {
          "path": {
            "prefix": "/"
          }
        },
        "fault": {
          "percentage": 100,
          "delay": {
            "fixed_delay_ms": 500
          }
        }
      }
    ]
  }
}
```

### Response

```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "status": "success",
  "message": "Policy delay-demo created successfully"
}
```

## Breaking Changes

| Field | Old | New | Migration |
|-------|-----|-----|-----------|
| `fault.delay.fixed_delay` | `string` | Removed | Use `fixed_delay_ms` |
| `fault.delay.fixed_delay_ms` | N/A | `integer` | New field (required) |

## Validation Rules

1. `fixed_delay_ms` MUST be a non-negative integer
2. `fixed_delay_ms` SHOULD be <= 30000 (values above will be clamped)
3. `fixed_delay_ms = 0` is treated as no delay (fault not injected)
