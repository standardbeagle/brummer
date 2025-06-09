---
sidebar_position: 4
---

# Microservices Development

Master microservices development with Brummer's powerful process orchestration and monitoring capabilities.

## Overview

Microservices architecture introduces complexity:
- Multiple services running on different ports
- Inter-service communication
- Service discovery
- Distributed logging
- Health monitoring
- Database per service

Brummer simplifies managing this complexity with unified monitoring and control.

## Example Microservices Architecture

```
microservices-app/
├── services/
│   ├── api-gateway/      # Kong/Express gateway
│   ├── auth-service/     # Authentication (JWT)
│   ├── user-service/     # User management
│   ├── product-service/  # Product catalog
│   ├── order-service/    # Order processing
│   ├── payment-service/  # Payment processing
│   └── notification/     # Email/SMS service
├── shared/
│   ├── logger/          # Centralized logging
│   └── types/           # Shared TypeScript types
├── docker-compose.yml
└── package.json
```

## Root Package Configuration

```json title="package.json (root)"
{
  "name": "microservices-app",
  "scripts": {
    // Individual services
    "gateway": "cd services/api-gateway && npm run dev",
    "auth": "cd services/auth-service && npm run dev",
    "users": "cd services/user-service && npm run dev",
    "products": "cd services/product-service && npm run dev",
    "orders": "cd services/order-service && npm run dev",
    "payments": "cd services/payment-service && npm run dev",
    "notifications": "cd services/notification && npm run dev",
    
    // Infrastructure
    "db:postgres": "docker-compose up -d postgres",
    "db:mongo": "docker-compose up -d mongodb",
    "redis": "docker-compose up -d redis",
    "rabbitmq": "docker-compose up -d rabbitmq",
    
    // Orchestration
    "dev": "concurrently -n gateway,auth,users,products,orders \"npm:gateway\" \"npm:auth\" \"npm:users\" \"npm:products\" \"npm:orders\"",
    "infra": "docker-compose up -d postgres mongodb redis rabbitmq",
    "logs": "docker-compose logs -f",
    
    // Testing
    "test:integration": "jest --config=jest.integration.config.js",
    "test:e2e": "wait-on tcp:3000 && jest --config=jest.e2e.config.js",
    
    // Monitoring
    "monitor": "node scripts/health-check.js"
  }
}
```

## Starting Your Microservices

```bash
cd microservices-app
brum
```

![Microservices Scripts](../img/screenshots/microservices-scripts.png)

### Step 1: Start Infrastructure

First, start the required infrastructure:

```bash
# In Brummer, run:
# 1. infra (starts all databases and message queues)
# 2. Check Processes tab for status
```

![Infrastructure Running](../img/screenshots/microservices-infra.png)

### Step 2: Start Core Services

Start services in dependency order:

1. **Auth Service** (no dependencies)
2. **User Service** (depends on auth)
3. **Product Service** (independent)
4. **API Gateway** (depends on all services)

![Services Starting](../img/screenshots/microservices-starting.png)

## Service Communication Monitoring

### Request Flow Tracking

Monitor inter-service communication:

```javascript title="services/api-gateway/src/middleware/logging.js"
app.use((req, res, next) => {
  const requestId = uuid();
  req.id = requestId;
  
  console.log(`[${requestId}] ${req.method} ${req.path} - Start`);
  
  res.on('finish', () => {
    console.log(`[${requestId}] ${req.method} ${req.path} - ${res.statusCode} (${Date.now() - req.startTime}ms)`);
  });
  
  next();
});
```

Brummer shows correlated logs:

```
[a1b2c3] POST /api/orders - Start
[auth-service] Validating token for request a1b2c3
[user-service] Loading user profile for request a1b2c3
[order-service] Creating order for request a1b2c3
[payment-service] Processing payment for order 12345
[a1b2c3] POST /api/orders - 201 (523ms)
```

### Service Health Monitoring

![Service Health](../img/screenshots/microservices-health.png)

Each service exposes health endpoints:

```javascript title="Health check standard"
app.get('/health', async (req, res) => {
  const health = {
    status: 'healthy',
    service: 'user-service',
    timestamp: new Date(),
    uptime: process.uptime(),
    memory: process.memoryUsage(),
    dependencies: {
      database: await checkDatabase(),
      redis: await checkRedis()
    }
  };
  
  res.json(health);
});
```

## Error Handling Across Services

### Distributed Error Tracking

When errors occur across services:

![Distributed Errors](../img/screenshots/microservices-errors.png)

Example error flow:
```
1. Payment service fails
2. Order service catches error
3. API Gateway returns user-friendly error
4. All logged with correlation ID
```

### Circuit Breaker Pattern

Monitor circuit breaker status:

```javascript
console.log(`[Circuit Breaker] Payment service circuit opened - failing fast`);
console.log(`[Circuit Breaker] Attempting reset after cooldown`);
console.log(`[Circuit Breaker] Payment service circuit closed - normal operation`);
```

## Message Queue Integration

### RabbitMQ Message Flow

Monitor asynchronous message processing:

```javascript title="Order processing flow"
// Order Service
console.log(`Publishing order.created event: ${orderId}`);

// Notification Service
console.log(`Received order.created event: ${orderId}`);
console.log(`Sending order confirmation email to ${userEmail}`);

// Inventory Service
console.log(`Received order.created event: ${orderId}`);
console.log(`Updating inventory for products: ${productIds}`);
```

![Message Queue Flow](../img/screenshots/microservices-rabbitmq.png)

## Database Per Service

### Managing Multiple Databases

Each service has its own database:

| Service | Database | Port |
|---------|----------|------|
| Users | PostgreSQL | 5432 |
| Products | MongoDB | 27017 |
| Orders | PostgreSQL | 5433 |
| Notifications | Redis | 6379 |

Monitor database connections:

![Database Connections](../img/screenshots/microservices-databases.png)

## Development Workflows

### 1. Developing a Single Service

Focus on one service:

```bash
# Traditional approach:
cd services/user-service
npm run dev
# Need separate terminals for logs, tests, etc.

# With Brummer:
brum -d services/user-service
# Everything in one interface
```

### 2. Integration Testing

Run integration tests with all services:

```json
{
  "scripts": {
    "test:integration:setup": "npm run infra && npm run dev",
    "test:integration:run": "wait-port 3000 3001 3002 && jest",
    "test:integration": "npm-run-all test:integration:*"
  }
}
```

Monitor test execution:

![Integration Tests](../img/screenshots/microservices-integration-tests.png)

### 3. Debugging Service Communication

Use Brummer's filtering to trace requests:

```bash
# Filter by request ID
/show a1b2c3

# Show only errors
/show ERROR

# Show specific service
/show user-service
```

## Performance Monitoring

### Service Metrics

Track key metrics per service:

```javascript
setInterval(() => {
  console.log(JSON.stringify({
    service: 'order-service',
    metrics: {
      requests_per_second: getCurrentRPS(),
      average_response_time: getAvgResponseTime(),
      active_connections: getActiveConnections(),
      memory_usage: process.memoryUsage().heapUsed / 1024 / 1024,
      cpu_usage: process.cpuUsage()
    }
  }));
}, 10000);
```

![Service Metrics](../img/screenshots/microservices-metrics.png)

### Load Testing

Monitor services under load:

```bash
# Run load test
npm run loadtest

# In Brummer, watch:
# - Response times increasing
# - Memory usage growing
# - Error rates
# - Circuit breakers activating
```

## Container Integration

### Docker Compose Development

```yaml title="docker-compose.yml"
version: '3.8'
services:
  api-gateway:
    build: ./services/api-gateway
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=development
    volumes:
      - ./services/api-gateway:/app
    command: npm run dev
```

Run containerized services:

```json
{
  "scripts": {
    "docker:dev": "docker-compose up",
    "docker:logs": "docker-compose logs -f",
    "docker:ps": "docker-compose ps"
  }
}
```

## Advanced Patterns

### Service Discovery

Monitor service registration:

```
[Consul] Registering user-service at 172.20.0.5:3001
[Consul] Registering order-service at 172.20.0.6:3002
[API Gateway] Discovered services: user-service, order-service
```

### Distributed Tracing

Track requests across services:

```
[Trace: 5f3a2b1c] Start: API Gateway
[Trace: 5f3a2b1c] → Auth Service (12ms)
[Trace: 5f3a2b1c] → User Service (45ms)
[Trace: 5f3a2b1c] → Order Service (78ms)
[Trace: 5f3a2b1c] → Payment Service (234ms)
[Trace: 5f3a2b1c] Complete: 369ms total
```

### Rate Limiting

Monitor rate limit enforcement:

```
[Rate Limit] Client 192.168.1.100 - 45/60 requests
[Rate Limit] Client 192.168.1.100 - 60/60 requests (limit reached)
[Rate Limit] Client 192.168.1.100 - Request blocked (429)
```

## Troubleshooting

### Common Issues

1. **Service Won't Start**
   - Check port availability
   - Verify database connections
   - Check environment variables

2. **Inter-Service Communication Fails**
   - Verify service discovery
   - Check network connectivity
   - Monitor timeout settings

3. **Message Queue Congestion**
   - Monitor queue depths
   - Check consumer health
   - Scale consumers if needed

### Debugging Tools

Use Brummer's features for debugging:

1. **Process Memory Monitoring**
   - Watch for memory leaks
   - Identify heavy services
   - Plan scaling needs

2. **Log Correlation**
   - Filter by request ID
   - Track error propagation
   - Identify bottlenecks

3. **Health Check Dashboard**
   - Custom script to check all services
   - Display in Brummer logs
   - Alert on failures

## Best Practices

### 1. Service Startup Order

```json
{
  "scripts": {
    "start:1:infra": "npm run infra",
    "start:2:core": "npm run auth users",
    "start:3:services": "npm run products orders payments",
    "start:4:gateway": "npm run gateway",
    "start:all": "npm-run-all start:*"
  }
}
```

### 2. Graceful Shutdown

Handle shutdown properly:

```javascript
process.on('SIGTERM', async () => {
  console.log('Shutting down gracefully...');
  await closeDatabase();
  await closeMessageQueue();
  server.close(() => {
    console.log('Service stopped');
    process.exit(0);
  });
});
```

### 3. Environment Management

```bash
# Development
NODE_ENV=development brum

# Staging
NODE_ENV=staging brum

# Local production testing
NODE_ENV=production brum
```

## Next Steps

- Learn about [Performance Monitoring](./performance-monitoring)
- Explore [CI Integration](./ci-integration)
- Set up [Team Collaboration](../tutorials/team-collaboration)