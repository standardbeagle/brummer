---
sidebar_position: 2
---

# Next.js Full-Stack Development

Master full-stack Next.js development with Brummer's intelligent process management and monitoring capabilities.

## Overview

Next.js applications often require managing:
- Development server with Fast Refresh
- API routes and serverless functions
- Database connections
- Background jobs
- Real-time features (WebSockets)

Brummer provides a unified interface for managing these complex workflows.

## Project Setup

### Full-Stack Next.js Configuration

```json title="package.json"
{
  "name": "nextjs-fullstack-app",
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "db:push": "prisma db push",
    "db:studio": "prisma studio",
    "db:seed": "tsx prisma/seed.ts",
    "workers": "tsx workers/process-jobs.ts",
    "websocket": "tsx servers/websocket.ts",
    "stripe:webhook": "stripe listen --forward-to localhost:3000/api/webhooks/stripe",
    "email:preview": "email dev --port 3001",
    "test": "jest --watch",
    "test:e2e": "playwright test",
    "analyze": "ANALYZE=true next build"
  }
}
```

### Starting Your Development Environment

```bash
cd nextjs-fullstack-app
brum
```

![Next.js Scripts Overview](../img/screenshots/nextjs-scripts.png)

## Development Workflow

### 1. Database Setup

Start with database initialization:

```bash
# In Brummer:
# 1. Run db:push to sync schema
# 2. Run db:seed to populate data
# 3. Run db:studio for visual management
```

![Database Processes](../img/screenshots/nextjs-database.png)

### 2. Core Development Services

Launch your main development services:

1. **Next.js Dev Server** (`dev`)
   - Fast Refresh enabled
   - API routes active
   - TypeScript checking

2. **Database Studio** (`db:studio`)
   - Visual database management
   - Real-time data updates

3. **Background Workers** (`workers`)
   - Job processing
   - Scheduled tasks

![Multiple Services Running](../img/screenshots/nextjs-services.png)

## Advanced Scenarios

### API Route Development

Monitor API routes in real-time:

```typescript title="app/api/users/route.ts"
export async function POST(request: Request) {
  console.log('Creating user...')
  
  try {
    const data = await request.json()
    const user = await prisma.user.create({ data })
    
    console.log('User created:', user.id)
    return Response.json(user)
  } catch (error) {
    console.error('Failed to create user:', error)
    return Response.json({ error }, { status: 500 })
  }
}
```

Brummer shows:
- Request logs with timing
- Database queries
- Error stack traces
- Response status codes

### Real-Time Features with WebSockets

Running WebSocket server alongside Next.js:

![WebSocket Integration](../img/screenshots/nextjs-websocket.png)

**Process Organization:**
- Main app on port 3000
- WebSocket server on port 3002
- Both processes monitored together

### Webhook Development

Developing with external webhooks (e.g., Stripe):

```bash
# Traditional approach:
# Terminal 1: npm run dev
# Terminal 2: stripe listen --forward-to localhost:3000/api/webhooks/stripe
# Terminal 3: Check logs in both terminals

# With Brummer:
# All logs centralized and searchable
```

![Webhook Testing](../img/screenshots/nextjs-webhooks.png)

## Error Handling and Debugging

### Server-Side Errors

Brummer captures and highlights server errors:

```typescript
// API route error example
export async function GET() {
  throw new Error('Database connection failed')
}
```

**Brummer Display:**
- 🔴 Error highlighted in red
- 📍 Stack trace with file locations
- 🕒 Timestamp for error occurrence
- 📋 Copy button for quick sharing

### Build Errors

Next.js build errors are clearly shown:

![Build Error](../img/screenshots/nextjs-build-error.png)

Common errors detected:
- Missing environment variables
- Import errors
- Type mismatches
- Build optimization issues

### Environment Variable Management

Monitor missing environment variables:

```bash
# Brummer shows:
⚠️  Missing required environment variables:
   - DATABASE_URL
   - NEXTAUTH_SECRET
   - STRIPE_SECRET_KEY
```

## Performance Monitoring

### Build Performance

Track build times and optimizations:

```bash
# Run analyze script
# Brummer shows bundle analysis URL
```

![Bundle Analysis](../img/screenshots/nextjs-bundle.png)

### Development Server Performance

Monitor Fast Refresh performance:
- Page compilation time
- Hot reload speed
- Memory usage
- CPU utilization

## Database Workflows

### Prisma Integration

Efficient database development workflow:

1. **Schema Changes**
   ```prisma
   model User {
     id        String   @id @default(cuid())
     email     String   @unique
     posts     Post[]
     createdAt DateTime @default(now())
   }
   ```

2. **Apply Changes**
   - Run `db:push` in Brummer
   - Monitor migration output
   - Check for errors

3. **Verify with Studio**
   - Keep `db:studio` running
   - See schema updates in real-time

![Prisma Workflow](../img/screenshots/nextjs-prisma.png)

## Testing Strategies

### Unit and Integration Tests

Run tests alongside development:

```typescript title="__tests__/api/users.test.ts"
describe('/api/users', () => {
  it('creates a user', async () => {
    const res = await POST('/api/users', {
      email: 'test@example.com'
    })
    
    expect(res.status).toBe(201)
  })
})
```

Brummer shows:
- Test results in real-time
- Failed test details
- Coverage information

### E2E Testing

Running Playwright tests:

![E2E Tests](../img/screenshots/nextjs-e2e.png)

Tips:
- Run E2E tests in separate process
- Monitor browser automation logs
- Track test execution time

## Production-like Development

### Running Production Build Locally

```bash
# Build and start production server
# 1. Run 'build' script
# 2. Run 'start' script
# 3. Test with production optimizations
```

Monitor:
- Build size warnings
- Performance metrics
- API route response times

## Advanced Tips

### 1. Process Groups

Organize related processes:

```json
{
  "scripts": {
    "dev:all": "concurrently \"npm:dev\" \"npm:workers\" \"npm:websocket\"",
    "test:all": "concurrently \"npm:test\" \"npm:test:e2e\""
  }
}
```

### 2. Log Filtering for API Routes

Filter API route logs:

```bash
# Show only API logs
/show api/

# Hide static asset requests
/hide _next/static

# Show only error responses
/show "status: [4-5]"
```

### 3. Memory Leak Detection

Watch for memory growth:
- Monitor process memory in Processes tab
- Set up alerts for threshold
- Restart processes when needed

## Integration Examples

### With Tailwind CSS

Monitor Tailwind compilation:
- JIT compilation logs
- CSS file size
- Unused style warnings

### With tRPC

Track tRPC procedure calls:
- Request/response logs
- Type validation errors
- Performance metrics

### With NextAuth

Monitor authentication flow:
- Session creation
- Provider callbacks
- JWT generation

## Troubleshooting

### Common Issues

1. **Port Conflicts**
   - Use different ports for each service
   - Brummer shows which process uses which port

2. **Database Connection Issues**
   - Check DATABASE_URL in logs
   - Verify database is running
   - Monitor connection pool

3. **Build Failures**
   - Check for TypeScript errors
   - Verify all dependencies installed
   - Look for missing env vars

## Best Practices

1. **Start Services in Order**
   ```
   1. Database
   2. Next.js dev server
   3. Background workers
   4. WebSocket server
   ```

2. **Use Environment Modes**
   ```json
   {
     "scripts": {
       "dev": "NODE_ENV=development next dev",
       "dev:staging": "NODE_ENV=staging next dev"
     }
   }
   ```

3. **Monitor Resource Usage**
   - Set memory limits for processes
   - Restart heavy processes periodically
   - Use production builds for performance testing

## Next Steps

- Learn about [Monorepo Workflows](./monorepo-workflows)
- Explore [Microservices Development](./microservices)
- Set up [CI Integration](./ci-integration)