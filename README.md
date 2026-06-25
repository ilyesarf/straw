# Compress big infra into one md file

### Compression Rates
- Logs: 3,774 lines -> 37 patterns (99.0% reduction)
- Topology: 746 edges -> 11 flows (98.5% reduction)
- Metrics: 507 samples -> 15 elevated (97.0% reduction)
- **Total LLM Token Reduction: 276k Tokens (Raw) -> 1100 Tokens (with straw) (99.5% reduction)**

### Test

```bash
$ go run main.go stream.txt
```