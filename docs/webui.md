# Web UI Guide

Vue.js-based single-page application for FlightlessSomething.

## Features

- Vue.js 3 with Composition API
- Client-side routing (no page reloads)
- Dark theme (Bootstrap 5)
- Chart visualization (Highcharts)
- Fast development with Vite

## Development

### Prerequisites

- Node.js 18+
- npm or yarn

### Setup

```bash
cd web
npm install
```

### Development Server

Start with hot-reload:

```bash
npm run dev
```

Runs on http://localhost:3000, proxies API to http://localhost:5000.

**Important**: Start Go backend first.

### Build for Production

```bash
npm run build
```

Output in `web/dist/` directory (embedded into Go binary).

## Project Structure

```
web/
├── package.json          # Dependencies
├── vite.config.js        # Vite config
├── index.html            # Entry point
├── src/
│   ├── main.js          # App initialization
│   ├── App.vue          # Root component
│   ├── router/
│   │   └── index.js     # Routes
│   ├── stores/
│   │   └── auth.js      # Auth state (Pinia)
│   ├── components/
│   │   ├── Navbar.vue          # Navigation
│   │   └── BenchmarkCharts.vue # Chart rendering
│   ├── views/
│   │   ├── Login.vue           # Login page
│   │   ├── Benchmarks.vue      # List page
│   │   ├── BenchmarkDetail.vue # Detail page
│   │   └── MyBenchmarks.vue    # User's benchmarks
│   ├── workers/
│   │   └── benchmarkCalculations.worker.js # Web Worker for calculations
│   ├── utils/
│   │   ├── dateFormatter.js    # Date helpers
│   │   └── workerManager.js    # Web Worker manager
│   └── api/
│       └── client.js    # API client
├── tests/
│   ├── dateFormatter.test.js  # Unit tests
│   └── *.spec.js              # E2E tests
└── dist/                # Build output (gitignored)
```

## API Integration

Web UI communicates via API client (`src/api/client.js`).

During development, Vite proxies API requests to backend.

Example:
```javascript
import { api } from '@/api/client';

// Get benchmarks
const benchmarks = await api.getBenchmarks(page, perPage);

// Create benchmark
const formData = new FormData();
formData.append('title', 'Game Title');
formData.append('files', file);
await api.createBenchmark(formData);
```

## State Management

Pinia store in `stores/auth.js` handles:
- User login (Discord OAuth and admin)
- Logout
- Authentication state

Example:
```javascript
import { useAuthStore } from '@/stores/auth';

const authStore = useAuthStore();
await authStore.login(username, password);
if (authStore.isAuthenticated) {
  // User is logged in
}
```

## Routing

Vue Router configuration:
- `/` - Redirects to `/benchmarks`
- `/benchmarks` - List all benchmarks
- `/benchmarks/:id` - Benchmark details
- `/my-benchmarks` - User's benchmarks
- `/login` - Login page

## Styling

Bootstrap 5 dark theme with scoped component styles.

Global styles in `src/App.vue`.

## Build Process

Full build with Make:
```bash
# From project root
make build
```

This:
1. Installs npm dependencies
2. Builds Vue.js app
3. Embeds in Go binary

Or manual:
```bash
cd web
npm run build
cd ..
go build -o server ./cmd/server
```

## Development Workflow

### Frontend Development

Terminal 1 - Backend:
```bash
go run ./cmd/server [options...]
```

Terminal 2 - Frontend:
```bash
cd web
npm run dev
```

Visit http://localhost:3000 for hot-reload.

### Backend Development

When working only on backend:
```bash
go run ./cmd/server [options...]
```

Access at http://localhost:5000.

## Testing

### Unit Tests

```bash
cd web
npm run test:unit
```

Or directly:
```bash
node tests/dateFormatter.test.js
```

### E2E Tests (Playwright)

```bash
cd web

# First time
npm install
npx playwright install --with-deps chromium

# Run tests
npm test

# With UI
npm run test:ui

# Headed mode
npm run test:headed
```

### Linting

```bash
cd web
npm run lint
npm run lint:fix  # Auto-fix
```

## Deployment

Web UI is embedded in Go binary. Deploy the single `server` binary.

The Go server:
1. Serves embedded assets
2. Handles API requests
3. Manages authentication

## Dark Reader Compatibility

The app includes `<meta name="color-scheme" content="dark">` to prevent browser extensions like Dark Reader from interfering with the native dark theme.

## Best Practices

1. **Components** - Keep components focused and reusable
2. **API calls** - Use `api/client.js` methods
3. **State** - Use Pinia stores for shared state
4. **Styling** - Use Bootstrap utilities, scope component styles
5. **Testing** - Add tests for new utilities and pages

## Performance Optimizations

### Multithreaded Benchmark Rendering

For large benchmark datasets (up to 1 million data points), the Web UI uses Web Workers to offload CPU-intensive calculations from the main thread, keeping the UI responsive.

**Architecture:**
- **Web Worker** (`src/workers/benchmarkCalculations.worker.js`) - Runs statistical calculations in background thread
- **Worker Manager** (`src/utils/workerManager.js`) - Promise-based API for worker communication
- **BenchmarkCharts Component** - Delegates heavy computations to worker

**What runs in the Web Worker:**
- Statistical calculations (averages, percentiles, standard deviation, variance)
- FPS calculations using harmonic mean
- Outlier filtering
- Density chart data generation
- Line chart data decimation (reducing 1M points to 2K for rendering)

**Benefits:**
- **Non-blocking UI** - Calculations run in separate thread, UI stays responsive
- **Faster rendering** - Pre-calculated stats and decimated data
- **Better UX** - Loading indicators show progress, no frozen interface

**How it works:**
1. Benchmark data arrives from backend
2. `calculateStatistics()` sends data to Web Worker
3. Worker calculates all statistics in parallel
4. Results returned via Promise
5. Charts render using pre-calculated data

**Example usage:**
```javascript
// Worker manager is singleton, initialized automatically
const result = await workerManager.calculateAll(dataArrays)
// result contains: fpsStats, frametimeStats, summaryStats, decimatedData
```

**Performance impact:**
- Large datasets (100K+ points): 10-50x improvement in UI responsiveness
- Main thread free for user interactions during calculations
- Reduced memory usage via data decimation

## Troubleshooting

### Vite dev server won't start
- Check Node.js version (18+)
- Delete `node_modules` and reinstall
- Check port 3000 not in use

### API calls fail in dev
- Ensure backend running on port 5000
- Check Vite proxy config in `vite.config.js`

### Build errors
- Run `npm install` first
- Check for TypeScript errors (if using TS)
- Clear `dist/` and rebuild

### Hot-reload not working
- Check file saved properly
- Restart Vite dev server
- Check browser console for errors
