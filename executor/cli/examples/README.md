# Policy Examples

This directory contains example fault injection policies that demonstrate various features of HFI.

## Available Examples

### Basic Fault Types

- `abort-policy.yaml` - Simple abort fault (HTTP 503)
  - Matches: `GET /`
  - Effect: Returns HTTP 503 status with 100% probability

- `delay-policy.yaml` - Simple delay fault (1 second)
  - Matches: `GET /test`
  - Effect: Adds 1000ms delay with 100% probability

### Advanced Examples

- `50-percent-policy.yaml` - Probability-based fault injection
  - Demonstrates 50% fault injection probability

- `header-policy.yaml` - Header-based fault matching
  - Shows how to match requests based on HTTP headers

- `no-fault-policy.yaml` - Policy with no faults (passthrough)
  - Useful for testing policy application without actual fault injection

## Usage

Apply a policy using the HFI CLI:

```bash
# Apply a policy
hfi-cli policy apply -f examples/delay-policy.yaml

# List applied policies
hfi-cli policy list

# Delete a policy
hfi-cli policy delete test-delay-policy
```

## Policy Structure

All policies follow the same basic structure:

```yaml
metadata:
  name: "policy-name"
  version: "1.0.0"  # Optional
spec:
  rules:
    - match:
        method:
          exact: "GET"
        path:
          prefix: "/api"
        headers:  # Optional
          - name: "x-user-type"
            exact: "premium"
      fault:
        percentage: 50  # 0-100
        delay:  # Optional
          fixed_delay: "1s"
        abort:  # Optional
          httpStatus: 503
```

## Testing Policies

After applying a policy, you can test it using curl:

```bash
# Test delay policy
time curl http://localhost:8000/test

# Test abort policy  
curl -w "HTTP %{http_code}\n" http://localhost:8000/

# Test header-based policy
curl -H "x-user-type: premium" http://localhost:8000/api/users
```

## Best Practices

1. Start with low percentages - Begin with 10-20% fault injection rates
2. Use meaningful names - Policy names should describe their purpose
3. Test in staging first - Always validate policies in non-production environments
4. Monitor metrics - Use Envoy admin interface to monitor fault injection metrics
5. Clean up policies - Remove test policies when no longer needed

## Troubleshooting

If a policy doesn't seem to be working:

1. Check that the policy was applied successfully: `hfi-cli policy list`
2. Verify the request matches the policy conditions (method, path, headers)
3. Check Envoy logs for any WASM plugin errors
4. Verify the control plane is healthy: `kubectl get pods -l app=hfi-control-plane`
