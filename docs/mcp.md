# MCP Server

FlightlessSomething includes a built-in [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server that allows AI assistants to interact with your benchmark data.

## Endpoint

```
POST /mcp
```

The MCP server uses the **Streamable HTTP** transport with JSON-RPC 2.0 protocol.

## Authentication

The MCP server supports two modes:

- **Anonymous (read-only)**: Connect without an API token to browse and search benchmarks
- **Authenticated (read-write)**: Provide an API token to create, update, and delete benchmarks

Authentication uses the same API tokens as the REST API. Include your token in the `Authorization` header:

```
Authorization: Bearer YOUR_API_TOKEN
```

API tokens can be created from the web interface under **API Tokens**.

## Setup

### Visual Studio Code

1. Open your workspace in VS Code
2. Create `.vscode/mcp.json` in your workspace root (or add to user settings under `"mcp"`):

```json
{
  "servers": {
    "FlightlessSomething": {
      "type": "http",
      "url": "https://your-server.com/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_API_TOKEN"
      }
    }
  }
}
```

Alternatively, use the **Install in VS Code** button on the [API Tokens](/api-tokens) page to configure automatically.

### Claude Desktop

Add to your `claude_desktop_config.json` (typically at `~/.config/claude/claude_desktop_config.json` on Linux or `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "FlightlessSomething": {
      "type": "http",
      "url": "https://your-server.com/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_API_TOKEN"
      }
    }
  }
}
```

### Other MCP Clients

Any MCP client that supports the **Streamable HTTP** transport can connect. Configure it with:

- **URL**: `https://your-server.com/mcp`
- **Transport**: HTTP (Streamable HTTP)
- **Headers**: `Authorization: Bearer YOUR_API_TOKEN` (optional, for write access)

## Available Tools

### Read-Only Tools (no authentication required)

#### `list_benchmarks`

Search and list benchmarks with pagination, search, and sorting.

| Parameter    | Type    | Description                                      |
|-------------|---------|--------------------------------------------------|
| `page`      | integer | Page number (default: 1)                          |
| `per_page`  | integer | Results per page, 1-100 (default: 10)             |
| `search`    | string  | Search keywords (space-separated, AND logic)      |
| `user_id`   | integer | Filter by user ID                                 |
| `sort_by`   | string  | `title`, `created_at`, or `updated_at`            |
| `sort_order`| string  | `asc` or `desc` (default: `desc`)                 |

#### `get_benchmark`

Get detailed information about a specific benchmark.

| Parameter | Type    | Required | Description  |
|----------|---------|----------|--------------|
| `id`     | integer | Yes      | Benchmark ID |

#### `get_benchmark_data`

Get computed statistics for all benchmark runs. By default returns **stats only** (no raw data points). Stats match what the web UI displays: min, max, avg, median, p01, p97, std_dev, variance, count. FPS stats are correctly derived from frametime data.

| Parameter    | Type    | Required | Description                                                    |
|-------------|---------|----------|----------------------------------------------------------------|
| `id`        | integer | Yes      | Benchmark ID                                                    |
| `max_points`| integer | No       | Include downsampled raw data points (default: 0 = stats only). Set 1-5000 to include time series data alongside stats. |

#### `get_benchmark_run`

Get computed statistics for a specific run within a benchmark. Same format as `get_benchmark_data` but for a single run.

| Parameter    | Type    | Required | Description                     |
|-------------|---------|----------|---------------------------------|
| `id`        | integer | Yes      | Benchmark ID                     |
| `run_index` | integer | Yes      | Run index (0-based)              |
| `max_points`| integer | No       | Include downsampled raw data (default: 0 = stats only) |

### Write Tools (authentication required)

#### `update_benchmark`

Update benchmark metadata and/or run labels. Only the benchmark owner or an admin can update.

| Parameter     | Type    | Required | Description                                        |
|--------------|---------|----------|----------------------------------------------------|
| `id`         | integer | Yes      | Benchmark ID                                        |
| `title`      | string  | No       | New title (max 100 chars)                           |
| `description`| string  | No       | New description (max 5000 chars)                    |
| `labels`     | object  | No       | Map of run index to new label, e.g. `{"0": "Run A"}`|

#### `delete_benchmark`

Delete a benchmark and all its data. Only the benchmark owner or an admin can delete.

| Parameter | Type    | Required | Description  |
|----------|---------|----------|--------------|
| `id`     | integer | Yes      | Benchmark ID |

#### `delete_benchmark_run`

Delete a specific run from a benchmark. Cannot delete the last remaining run.

| Parameter    | Type    | Required | Description        |
|-------------|---------|----------|--------------------|
| `id`        | integer | Yes      | Benchmark ID        |
| `run_index` | integer | Yes      | Run index (0-based) |

## Computed Statistics

The MCP server computes the same statistics as the web frontend, for every metric in every run. These stats are **always returned** (no raw data points needed).

### Per-Metric Statistics

| Field      | Description                                                    |
|-----------|----------------------------------------------------------------|
| `min`     | Minimum value                                                   |
| `max`     | Maximum value                                                   |
| `avg`     | Average (mean)                                                  |
| `median`  | 50th percentile (median)                                        |
| `p01`     | 1st percentile                                                  |
| `p97`     | 97th percentile                                                 |
| `std_dev` | Standard deviation (sample, n-1 divisor, matches Excel/LibreOffice) |
| `variance`| Variance (sample, n-1 divisor, matches Excel/LibreOffice)       |
| `count`   | Total number of data points                                     |

### FPS Calculation

FPS statistics are derived from frametime data (matching the web frontend), since averaging FPS values directly is statistically incorrect:

- **Average FPS** = 1000 / average frametime
- **Min FPS** = 1000 / max frametime
- **Max FPS** = 1000 / min frametime
- **P01 FPS** = 1000 / 99th percentile frametime
- **P97 FPS** = 1000 / 3rd percentile frametime

### Raw Data Points (Optional)

By default, only stats are returned. Set `max_points` > 0 to also include downsampled time series data:

- Evenly-spaced point selection preserves temporal patterns
- Maximum 5000 points per metric
- Raw data appears in the `data` field of each metric

### Available Metrics

| Metric           | Description              |
|-----------------|--------------------------|
| `fps`           | Frames per second         |
| `frame_time`    | Frame time (ms)           |
| `cpu_load`      | CPU load (%)              |
| `gpu_load`      | GPU load (%)              |
| `cpu_temp`      | CPU temperature (°C)      |
| `cpu_power`     | CPU power (W)             |
| `gpu_temp`      | GPU temperature (°C)      |
| `gpu_core_clock`| GPU core clock (MHz)      |
| `gpu_mem_clock` | GPU memory clock (MHz)    |
| `gpu_vram_used` | GPU VRAM used (MB)        |
| `gpu_power`     | GPU power (W)             |
| `ram_used`      | RAM used (MB)             |
| `swap_used`     | Swap used (MB)            |

## Security

The MCP server follows the exact same security model as the REST API:

- **Banned users** cannot authenticate
- **Ownership checks** ensure users can only modify their own benchmarks
- **Admin users** can modify any benchmark
- **API token tracking** records last usage timestamps
- **Rate limiting** applies to write operations (same as REST API)

## Protocol Details

The MCP server implements the [MCP Streamable HTTP transport](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http) with:

- **Protocol version**: `2025-03-26`
- **JSON-RPC 2.0** over HTTP POST
- **Supported methods**: `initialize`, `notifications/initialized`, `tools/list`, `tools/call`, `ping`
- **Content-Type**: `application/json`
