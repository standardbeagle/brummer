const express = require('express');
const app = express();
const port = 3333;

app.get('/', (req, res) => {
  console.log(`Request from ${req.ip} at ${new Date().toISOString()}`);
  res.send(`
    <html>
      <head><title>Simple Test</title></head>
      <body>
        <h1>Simple Test Server</h1>
        <p>Current time: ${new Date().toISOString()}</p>
        <p>This page should have telemetry injected when accessed through the proxy.</p>
      </body>
    </html>
  `);
});

app.listen(port, () => {
  console.log(`Simple test server running at http://localhost:${port}`);
});