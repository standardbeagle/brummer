import express, { Request, Response, NextFunction } from 'express';

const app = express();
const port = process.env.PORT || 3000;

app.use(express.json());

// Middleware with intentional errors
app.use((req: Request, res: Response, next: NextFunction) => {
  // Error: Accessing undefined property
  console.log((req as any).undefinedProperty.length);
  next();
});

// Route with TypeScript errors
app.get('/typescript-error', (req: Request, res: Response) => {
  // Error: Type 'string' is not assignable to type 'number'
  const numberVar: number = "invalid" as any;
  
  // Error: Property 'nonExistent' does not exist on type 'Request'
  console.log((req as any).nonExistent);
  
  res.json({ error: 'TypeScript error endpoint' });
});

// Route with runtime errors
app.get('/runtime-error', (req: Request, res: Response) => {
  try {
    // Error: Cannot read properties of null
    const nullVar = null;
    console.log((nullVar as any).someProperty);
    
    // Error: Cannot read properties of undefined
    const undefinedVar = undefined;
    console.log((undefinedVar as any).someMethod());
    
    // Error: ReferenceError
    console.log((global as any).undefinedGlobal);
    
  } catch (error) {
    console.error('Runtime error in /runtime-error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Route with async errors
app.get('/async-error', async (req: Request, res: Response) => {
  try {
    // Error: Network request to invalid URL
    const response = await fetch('https://invalid-api-endpoint.nonexistent');
    const data = await response.json();
    res.json(data);
  } catch (error) {
    console.error('Async error in /async-error:', error);
    res.status(500).json({ error: 'Failed to fetch external data' });
  }
});

// Route with database simulation errors
app.get('/database-error', async (req: Request, res: Response) => {
  try {
    // Simulate MongoDB connection error
    throw new Error('MongoError: getaddrinfo ENOTFOUND cluster0.mongodb.net');
  } catch (error) {
    console.error('Database error:', error);
    res.status(500).json({ error: 'Database connection failed' });
  }
});

// Route with validation errors
app.post('/validation-error', (req: Request, res: Response) => {
  const { email, password } = req.body;
  
  // Error: Missing required fields
  if (!email) {
    const error = new Error('ValidationError: email is required');
    console.error('Validation error:', error);
    return res.status(400).json({ error: error.message });
  }
  
  // Error: Invalid email format
  if (!email.includes('@')) {
    const error = new Error('ValidationError: invalid email format');
    console.error('Validation error:', error);
    return res.status(400).json({ error: error.message });
  }
  
  res.json({ message: 'Validation passed' });
});

// Route that throws unhandled errors
app.get('/unhandled-error', (req: Request, res: Response) => {
  // Error: Unhandled exception
  throw new Error('Unhandled server error');
});

// Route with promise rejection
app.get('/promise-rejection', (req: Request, res: Response) => {
  // Error: Unhandled promise rejection
  Promise.reject(new Error('Unhandled promise rejection in Express route'));
  
  res.json({ message: 'Response sent before promise rejection' });
});

// Route with file system errors
app.get('/file-error', (req: Request, res: Response) => {
  const fs = require('fs');
  
  try {
    // Error: ENOENT: no such file or directory
    const data = fs.readFileSync('/nonexistent/file.txt', 'utf8');
    res.json({ data });
  } catch (error) {
    console.error('File system error:', error);
    res.status(500).json({ error: 'File not found' });
  }
});

// Route with parsing errors
app.post('/parse-error', (req: Request, res: Response) => {
  try {
    // Error: Unexpected token in JSON
    const invalidJson = '{"invalid": json}';
    const parsed = JSON.parse(invalidJson);
    res.json(parsed);
  } catch (error) {
    console.error('JSON parse error:', error);
    res.status(400).json({ error: 'Invalid JSON format' });
  }
});

// Global error handler
app.use((error: Error, req: Request, res: Response, next: NextFunction) => {
  console.error('Global error handler caught:', error);
  console.error('Stack trace:', error.stack);
  
  if (!res.headersSent) {
    res.status(500).json({
      error: 'Internal Server Error',
      message: error.message,
      stack: process.env.NODE_ENV === 'development' ? error.stack : undefined
    });
  }
});

// Handle 404 errors
app.use('*', (req: Request, res: Response) => {
  const error = new Error(`Route ${req.originalUrl} not found`);
  console.error('404 error:', error);
  res.status(404).json({ error: 'Route not found' });
});

// Process error handlers
process.on('uncaughtException', (error: Error) => {
  console.error('Uncaught Exception:', error);
  console.error('Stack:', error.stack);
  process.exit(1);
});

process.on('unhandledRejection', (reason: any, promise: Promise<any>) => {
  console.error('Unhandled Rejection at:', promise);
  console.error('Reason:', reason);
});

app.listen(port, () => {
  console.log(`Express server running on port ${port}`);
  
  // Trigger some initial errors
  triggerStartupErrors();
});

function triggerStartupErrors() {
  // Error: Startup configuration error
  try {
    const config = JSON.parse('{"invalid": json}');
  } catch (error) {
    console.error('Startup configuration error:', error);
  }
  
  // Error: Environment variable access
  const requiredEnvVar = process.env.REQUIRED_VAR;
  if (!requiredEnvVar) {
    console.error('Environment Error: REQUIRED_VAR is not set');
  }
}

export default app;